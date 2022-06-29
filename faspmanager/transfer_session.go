package faspmanager

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

// SessionId uniquely identifies a FASP transfer started with a call to one of
// Manager's StartTransfer methods.
type SessionId string

// TransferSession contains information about an ongoing or completed transfer and
// is responsible for communicating with the ASCP process once a management
// connection has been established.
type transferSession struct {
	id            SessionId
	order         TransferOrder
	ascpCommand   *exec.Cmd
	isRemote      bool // Was this session initiated remotely?
	isPersistent  bool
	streamEnabled bool
	streamWorker  *streamWorker
	chunkSize     int
	isLocked      bool
	processes     int // Number of ASCP processes utilized for this transfer.
	processIndex  int // Process index of this particular session.
	mgmtConn      net.Conn
	mgmtWriteSync sync.Mutex
	msgBuffer     string
	listeners     []TransferListener
	listenersSync sync.Mutex
	stats         *SessionStatistics
	statsSync     sync.RWMutex
	fileStats     map[string]*FileStatistics
	fileStatsSync sync.Mutex

	// Closed internally when the session's management connection is set,
	// indicating ASCP has started the transfer.
	started chan struct{}

	// Carries the exit error returned by a call to exec.Wait() on the underlying
	// ASCP process.
	ascpErr chan error

	// Closed internally when the session receives a management message indicating
	// completion.
	complete chan struct{}

	// Closing this channel indicates that the session should terminate,
	// cleaning up the management connection and worker goroutines.
	terminate  chan struct{}
	onComplete func(SessionId)
}

type sessionOptions struct {
	id             SessionId
	ascpPath       string
	processes      int
	processIndex   int
	managementPort int
	terminate      chan struct{}
	onComplete     func(SessionId)
}

var (
	ErrNotStarted    = errors.New("TransferSession: Communication over transfer connection has not yet started. Try calling Manager.WaitOnSessionStart.")
	ErrComplete      = errors.New("TransferSession: Session already complete. Cannot perform action.")
	ErrNonPersistent = errors.New("TransferSession: Cannot perform action on non-persistent session.")
	ErrNonStream     = errors.New("TransferSession: Cannot perform action on non-streaming session.")
	ErrLocked        = errors.New("TransferSession: Session already locked. Cannot perform action.")
)

func newTransferSession(sessOpts sessionOptions, ascpOpts *AscpOptions) (*transferSession, error) {
	if ascpOpts == nil {
		return nil, errors.New("TransferSession: ascpOpts cannot be nil.")
	}

	cmd, err := buildAscpCommand(sessOpts, ascpOpts)
	if err != nil {
		return nil, err
	}

	transferOpts := ascpOpts.TransferOpts
	if transferOpts == nil {
		transferOpts = &TransferOptions{}
	}

	stats := &SessionStatistics{
		Id:             sessOpts.id,
		Encryption:     !transferOpts.DisableEncryption,
		Policy:         transferOpts.Policy,
		Token:          transferOpts.Token,
		MinRateKbps:    transferOpts.MinRateKbps,
		TransferRetry:  transferOpts.TransferRetry,
		TargetRateKbps: transferOpts.getTargetRateOrSetDefault(),
		UdpPort:        transferOpts.getUdpPortOrSetDefault(),
		TcpPort:        transferOpts.getTcpPortOrSetDefault(),
	}

	return &transferSession{
		id:            sessOpts.id,
		processes:     sessOpts.processes,
		processIndex:  sessOpts.processIndex,
		isPersistent:  ascpOpts.Persistent,
		streamEnabled: ascpOpts.Stream,
		chunkSize:     transferOpts.getChunkSizeOrSetDefault(),
		ascpCommand:   cmd,
		stats:         stats,
		mgmtWriteSync: sync.Mutex{},
		statsSync:     sync.RWMutex{},
		listenersSync: sync.Mutex{},
		fileStats:     make(map[string]*FileStatistics),
		fileStatsSync: sync.Mutex{},
		ascpErr:       make(chan error),
		started:       make(chan struct{}),
		complete:      make(chan struct{}),
		terminate:     sessOpts.terminate,
		onComplete:    sessOpts.onComplete,
	}, nil
}

func buildAscpCommand(sessOpts sessionOptions, ascpOpts *AscpOptions) (*exec.Cmd, error) {
	cmdLine, err := ascpOpts.buildAscpCommandLine()
	if err != nil {
		return nil, err
	}
	var args []string
	if sessOpts.processes > 1 {
		args = append(args, "-C")
		args = append(args, strconv.Itoa(sessOpts.processIndex)+":"+strconv.Itoa(sessOpts.processes))
	}

	args = append(args, "-M"+strconv.Itoa(sessOpts.managementPort))
	args = append(args, "-u"+string(sessOpts.id))
	args = append(args, cmdLine...)

	command := exec.Command(sessOpts.ascpPath, args...)

	command.Env = append(command.Env, os.Environ()...)
	if len(ascpOpts.RemotePass) != 0 {
		command.Env = append(command.Env, "ASPERA_SCP_PASS="+ascpOpts.RemotePass)
	}
	if len(ascpOpts.TransferOpts.Token) != 0 {
		command.Env = append(command.Env, "ASPERA_SCP_TOKEN="+ascpOpts.TransferOpts.Token)
	}
	if len(ascpOpts.TransferOpts.Cookie) != 0 {
		command.Env = append(command.Env, "ASPERA_SCP_COOKIE="+ascpOpts.TransferOpts.Cookie)
	}
	if len(ascpOpts.TransferOpts.ContentProtectionPassphrase) != 0 {
		command.Env = append(command.Env, "ASPERA_SCP_FILEPASS="+ascpOpts.TransferOpts.ContentProtectionPassphrase)
	}
	return command, nil
}

// start starts ASCP, initiating the transfer.
func (sess *transferSession) start() error {
	sess.stats.State = SUBMITTED_SESSION_STATE
	sess.stats.StartTime = time.Now()

	// Capture ASCP's standard error.
	ascpStdErr := new(bytes.Buffer)
	sess.ascpCommand.Stderr = ascpStdErr

	err := sess.ascpCommand.Start()
	if err != nil {
		return fmt.Errorf("TransferSession: Unable to start ASCP. Be sure the ASCP executable's location "+
			"has been explicitly set or can be found in the environment's path variable and that a valid "+
			"Aspera license file is present. Error: %v\n", err)
	}

	// Listen for ASCP's exit status and notify interested goroutines.
	go func(ascpCommand *exec.Cmd, ascpStdErr *bytes.Buffer, ascpErr chan<- error, terminate <-chan struct{}) {
		err := ascpCommand.Wait()
		if err != nil {
			err = fmt.Errorf("TransferSession: ASCP process exited with non-nil error: %v. "+
				"\n\nStandard error dump of ASCP process:\n\n %v\n", err, ascpStdErr.String())
			fmLog.Println(err)
		}
		select {
		case ascpErr <- err:
		case <-terminate:
		}
	}(sess.ascpCommand, ascpStdErr, sess.ascpErr, sess.terminate)

	return nil
}

// waitOnStart blocks until the session's management connection is set, indicating
// that the first management message has been received from ASCP, or until ASCP
// exits unexpectedly, in which case the corresponding error message is returned.
func (sess *transferSession) waitOnStart() error {
	select {
	case err := <-sess.ascpErr:
		return err
	case <-sess.started:
		return nil
	}
}

// addFileSource adds a file source to a persistent session.  Note that the session
// must be persistent, ongoing, and that TransferSession.LockPersistentSession
// cannot have preceded this call.
func (sess *transferSession) addFileSource(sourcePath string, destPath string, startByte, endByte int64) error {
	if !sess.isPersistent {
		return ErrNonPersistent
	}
	if !sess.isStarted() {
		return ErrNotStarted
	}
	if sess.isLocked {
		return ErrLocked
	}
	if sess.isComplete() {
		return ErrComplete
	}

	message := "FASPMGR 2\nType: START\nSource: " + sourcePath + "\nDestination: " + destPath + "\n"
	if startByte != 0 || endByte != 0 {
		message = message + "FileBytes: "
		if startByte != 0 {
			message = message + strconv.FormatInt(startByte, 10)
		}
		message = message + ":"
		if endByte != 0 {
			message = message + strconv.FormatInt(endByte, 10)
		}
		message = message + "\n"
	}
	message = message + "\n"
	return sess.sendManagementMessage(message)
}

// addReaderSource adds a reader source to a stream-enabled session
// (PersistentStreamUpload). The reader is consumed on a worker goroutine, with
// healthy completion signalled by the reader returning an io.EOF error.
// Termination of the session through its 'terminate' channel will stop the
// transfer prior to the next read.
func (sess *transferSession) addReaderSource(reader io.Reader, destPath string) error {
	err := sess.validateStreamReady()
	if err != nil {
		return err
	}
	sess.streamWorker.enqueue(readerJob{Source: reader, DestPath: destPath})
	return nil
}

// addStreamReaderSource adds a StreamReader source to a stream-enabled session.
// See transferSession.addReaderSource.
func (sess *transferSession) addStreamReaderSource(reader StreamReader, destPath string) error {
	err := sess.validateStreamReady()
	if err != nil {
		return err
	}

	sess.streamWorker.enqueue(streamReaderJob{Source: reader, DestPath: destPath})
	return nil
}

func (sess *transferSession) validateStreamReady() error {
	if !sess.streamEnabled {
		return ErrNonStream
	}
	if !sess.isStarted() {
		return ErrNotStarted
	}
	if sess.isLocked {
		return ErrLocked
	}
	if sess.isComplete() {
		return ErrComplete
	}
	return nil
}

func (sess *transferSession) notifyStreamErr(err StreamErr, destPath string) {
	sess.fileStatsSync.Lock()
	fStats, exists := sess.fileStats[destPath]
	sess.fileStatsSync.Unlock()
	if !exists {
		fStats = &FileStatistics{Name: destPath}
	}

	fStats.ErrorDescription = err.Error()
	sess.stats.ErrorDescription = err.Error()

	event := TransferEvent{
		EventType:    FILE_ERROR_TRANSFER_EVENT,
		SessionStats: sess.getStatsCopy(),
		FileStats:    fStats,
	}

	sess.listenersSync.Lock()
	defer sess.listenersSync.Unlock()
	for _, listener := range sess.listeners {
		listener.OnTransferEvent(event)
	}
}

// setRate dynamically sets the transfer rate and policy of this session.
func (sess *transferSession) setRate(targetRateKbps int, minRateKbps int, policy TransferPolicyType) error {
	if !sess.isStarted() {
		return ErrNotStarted
	}
	if sess.isLocked {
		return ErrLocked
	}
	if sess.isComplete() {
		return ErrComplete
	}

	policyStr, err := policy.getManagementMessageString()
	if err != nil {
		return err
	}

	message := "FASPMGR 2\nType: RATE\nRate: " + strconv.Itoa(targetRateKbps) +
		"\nMinRate: " + strconv.Itoa(minRateKbps) + "\nAdaptive: " + policyStr + "\n\n"
	return sess.sendManagementMessage(message)
}

// cancel cancels this session if it is ongoing or returns an error otherwise.
func (sess *transferSession) cancel() error {
	if !sess.isStarted() {
		return ErrNotStarted
	}
	if sess.isComplete() {
		return ErrComplete
	}
	if sess.isPersistent {
		return sess.sendManagementMessage("FASPMGR 2\nType: DONE\n\n")
	} else {
		return sess.sendManagementMessage("FASPMGR 2\nType: CANCEL\n\n")
	}
}

// lock "locks" a persistent session, disallowing the addition of sources with
// transferSession.addSource.
func (sess *transferSession) lock() error {
	if !sess.isPersistent && !sess.streamEnabled {
		return errors.New("Cannot lock a non-persistent, non-streaming session.")
	}
	if !sess.isStarted() {
		return ErrNotStarted
	}
	if sess.isLocked {
		return ErrLocked
	}
	if sess.isComplete() {
		return ErrComplete
	}

	sess.isLocked = true
	return sess.sendManagementMessage("FASPMGR 2\nType: DONE\nOperation: Linger\n\n")
}

// waitOnStop blocks until this session is complete or a terminal error has occurred.
func (sess *transferSession) waitOnStop() error {
	select {
	case err := <-sess.ascpErr:
		return err
	case <-sess.complete:
		return nil
	}
}

func (sess *transferSession) isStarted() bool {
	select {
	case <-sess.started:
		return true
	default:
		return false
	}
}

func (sess *transferSession) isComplete() bool {
	select {
	case <-sess.complete:
		return true
	default:
		return false
	}
}

// addListener adds a session-specific TransferListener to this session.
func (sess *transferSession) addListener(l TransferListener) {
	sess.listenersSync.Lock()
	sess.listeners = append(sess.listeners, l)
	sess.listenersSync.Unlock()
}

// removeListener removes a TransferListener from this session. Returns true if the given
// listener is found and removed, false otherwise.
func (sess *transferSession) removeListener(l TransferListener) bool {
	sess.listenersSync.Lock()
	defer sess.listenersSync.Unlock()
	for i, listener := range sess.listeners {
		if listener == l {
			sess.listeners = append(sess.listeners[:i], sess.listeners[i+1:]...)
			return true
		}
	}
	return false
}

// getStatsCopy returns a copy of this sessions' statistics.
func (sess *transferSession) getStatsCopy() *SessionStatistics {
	sess.statsSync.Lock()
	stats := *sess.stats
	sess.statsSync.Unlock()
	return &stats
}

// setManagementConn sets the connection through which this session communicates
// with ASCP.  This must be called if the session is to receive and send updates
// regarding its transfer(s).
func (sess *transferSession) setManagementConn(conn net.Conn) error {
	if sess.isStarted() {
		return errors.New("TransferSession: management connection has already been set.")
	}
	sess.mgmtConn = conn
	if sess.streamEnabled {
		sess.streamWorker = newStreamWorker(
			sess.mgmtConn,
			&sess.mgmtWriteSync,
			sess.chunkSize,
			sess.complete,
			sess.terminate,
			func(err StreamErr, dest string) { sess.notifyStreamErr(err, dest) },
		)
	}
	go sess.listenForManagementMessages(conn)
	close(sess.started)
	return nil
}

func (sess *transferSession) sendManagementMessage(message string) error {
	messageBytes := []byte(message)
	sess.mgmtWriteSync.Lock()
	defer sess.mgmtWriteSync.Unlock()
	for bytesWritten := 0; bytesWritten < len(messageBytes); {

		// No need to set a write deadline here - if the session terminates (<-sess.terminate),
		// transferSession.listenForManagementMessages has the responsibility of closing
		// the connection, causing any ongoing write to fail.
		n, err := sess.mgmtConn.Write(messageBytes[bytesWritten:])
		if err != nil {
			select {

			// Check if the error is because ASCP crashed.
			case ascpErr := <-sess.ascpErr:
				return fmt.Errorf("TransferSession: Unable to write to ASCP management connection. ASCP "+
					"process has already exited with error: %v.", ascpErr)
			default:
				return fmt.Errorf("TransferSession: IO Error writing to ASCP management connection: %+v\n", err)
			}
		}
		bytesWritten += n
	}
	return nil
}

func (sess *transferSession) listenForManagementMessages(mgmtConn net.Conn) {

	// This routine has the responsibility of closing the management connection if
	// the session terminates (<-sess.terminate) or if a read returns an io.EOF error.
	defer mgmtConn.Close()

	managementMessageBuffer := make([]byte, 2048)
	for {
		select {
		case <-sess.terminate:
			fmLog.Println("TransferSession: Terminating management TCP connection.")
			return
		default:
		}
		mgmtConn.SetReadDeadline(time.Now().Add(workerCloseDeadline))
		bytesRead, err := mgmtConn.Read(managementMessageBuffer)
		if err != nil {
			if err == io.EOF {
				// The connection has been closed by the remote host.
				// This should occur when the transfer is complete.
				return
			} else {
				// Either the read has timed out, there has been a network
				// error, or ASCP crashed.
				continue
			}
		}
		err = sess.receiveManagementBlob(string(managementMessageBuffer[:bytesRead]))
		if err != nil {
			// Ignore.
			fmLog.Println(err)
			continue
		}
	}
}

func (sess *transferSession) receiveManagementBlob(blob string) error {
	sess.msgBuffer = sess.msgBuffer + blob
	for {
		index := strings.Index(sess.msgBuffer, "\n\n")
		if index == -1 {
			break
		}
		currMsg := sess.msgBuffer[0:index]
		sess.msgBuffer = sess.msgBuffer[index+2:]
		if err := sess.processSingleMessage(currMsg); err != nil {
			return err
		}
	}
	return nil
}

func (sess *transferSession) processSingleMessage(message string) error {
	mgmtLog.Println("Session with ID " + string(sess.id) + " is processing management message:\n\n" + message)
	var msgType string
	msgFileStats := &FileStatistics{}
	msgLines := strings.Split(message, "\n")
	for _, line := range msgLines {
		if index := strings.Index(line, ":"); index != -1 {
			key, val := line[:index], line[index+2:]
			if key == "Type" {
				msgType = val
			}
			if err := sess.updateStatsFromMsg(key, val, msgFileStats); err != nil {
				return err
			}
		} else if !strings.Contains(line, "FASPMGR") {
			return errors.New("TransferSession: Management message contained unrecognized line: " +
				line + "\n\nMessage: " + message)
		}
	}

	// updatedStatsCopy will be nil if the message contained no file-specific info.
	updatedFileStatsCopy, wasNewFileCreated := sess.mergeFileStats(msgFileStats)

	eventType, err := getTransferEventType(msgType, wasNewFileCreated)
	if err != nil {
		return errors.New("TransferSession: Management message contained unrecognized 'Type' value. " +
			"Message: " + message)
	}

	if eventType == SESSION_START_TRANSFER_EVENT && sess.streamEnabled {
		sess.streamWorker.chunkSize = sess.chunkSize
		sess.streamWorker.start()
	}

	event := TransferEvent{
		EventType:    eventType,
		SessionStats: sess.getStatsCopy(),
		FileStats:    updatedFileStatsCopy,
	}

	sess.listenersSync.Lock()
	for _, listener := range sess.listeners {

		// All listeners receive a pointer to the same copy of SessionStats and FileStats.
		listener.OnTransferEvent(event)
	}
	sess.listenersSync.Unlock()

	if updatedFileStatsCopy != nil {
		fileState := updatedFileStatsCopy.State
		if fileState == SKIPPED_FILE_STATE ||
			fileState == FAILED_FILE_STATE ||
			fileState == FINISHED_FILE_STATE {

			sess.fileStatsSync.Lock()
			delete(sess.fileStats, updatedFileStatsCopy.Name)
			sess.fileStatsSync.Unlock()
		}
	}

	sess.statsSync.RLock()
	finished := sess.stats.State == FINISHED_SESSION_STATE || sess.stats.State == FAILED_SESSION_STATE
	sess.statsSync.RUnlock()
	if finished {
		if sess.onComplete != nil {
			sess.onComplete(sess.id)
		}
		select {

		// Check if sess.complete is already closed. This is possible because
		// ASCP sends multiple completion messages when HTTP fallback is used.
		case <-sess.complete:
		default:
			close(sess.complete)
		}
	}

	return nil
}

func getTransferEventType(msgType string, newFileObserved bool) (TransferEventType, error) {
	switch msgType {
	case "INIT":
		return CONNECTING_TRANSFER_EVENT, nil
	case "SESSION":
		return SESSION_START_TRANSFER_EVENT, nil
	case "NOTIFICATION":
		return RATE_MODIFICATION_TRANSFER_EVENT, nil
	case "STATS":
		if newFileObserved {
			return FILE_START_TRANSFER_EVENT, nil
		} else {
			return PROGRESS_TRANSFER_EVENT, nil
		}
	case "STOP":
		return FILE_STOP_TRANSFER_EVENT, nil
	case "FILEERROR":
		return FILE_ERROR_TRANSFER_EVENT, nil
	case "ERROR":
		return SESSION_ERROR_TRANSFER_EVENT, nil
	case "DONE":
		return SESSION_STOP_TRANSFER_EVENT, nil
	case "SKIP":
		return FILE_SKIP_TRANSFER_EVENT, nil
	default:
		return SESSION_ERROR_TRANSFER_EVENT, errors.New("TransferSession: Unrecognized management message type: " + msgType)
	}
}

// If statsFromMsg.Name is empty, makes no changes to this session and returns nil.
// Otherwise, updates the FileStatistics in sess.allFileStats with the same Name field as those given,
// or adds statsFromMsg to sess.allFileStats if no such stats exist. Returns a copy of the updated stats
// and whether the given stats were added to sess.allFileStats (true) or an existing member of sess.allFileStats
// was found with a matching Name field (false).
func (sess *transferSession) mergeFileStats(statsFromMsg *FileStatistics) (*FileStatistics, bool) {
	if statsFromMsg.Name == "" {
		return nil, false
	}
	var updatedStats *FileStatistics
	var newStatsCreated bool
	sess.fileStatsSync.Lock()
	defer sess.fileStatsSync.Unlock()
	existingStats, exists := sess.fileStats[statsFromMsg.Name]
	if exists {
		existingStats.ErrorCode = statsFromMsg.ErrorCode
		existingStats.State = statsFromMsg.State
		existingStats.FileChecksumType = statsFromMsg.FileChecksumType
		existingStats.FileChecksum = statsFromMsg.FileChecksum

		if statsFromMsg.ContiguousBytes != 0 {
			existingStats.ContiguousBytes = statsFromMsg.ContiguousBytes
		}
		if statsFromMsg.SizeBytes != 0 {
			existingStats.SizeBytes = statsFromMsg.SizeBytes
		}
		if statsFromMsg.StartByte != 0 {
			existingStats.StartByte = statsFromMsg.StartByte
		}
		if statsFromMsg.EndByte != 0 {
			existingStats.EndByte = statsFromMsg.EndByte
		}
		if statsFromMsg.WrittenBytes != 0 {
			existingStats.WrittenBytes = statsFromMsg.WrittenBytes
		}
		updatedStats = existingStats
		newStatsCreated = false
	} else {
		sess.fileStats[statsFromMsg.Name] = statsFromMsg
		updatedStats = statsFromMsg
		newStatsCreated = true
	}
	statsCopy := *updatedStats
	return &statsCopy, newStatsCreated
}

// updateStatsFromMsg updates this session's SessionStatistics and the given FileStatistics
// based on the given key value pair from a management message.
func (sess *transferSession) updateStatsFromMsg(key string, val string, tempFileStats *FileStatistics) error {
	sess.statsSync.Lock()
	defer sess.statsSync.Unlock()
	var parseErr error
	switch key {
	case "Adaptive":
		switch val {
		case "Adaptive":
			sess.stats.Policy = FAIR_TRANSFER_POLICY
		case "Fixed":
			sess.stats.Policy = FIXED_TRANSFER_POLICY
		case "Trickle":
			sess.stats.Policy = LOW_TRANSFER_POLICY
		default:
			return fmt.Errorf("TransferSession: Unrecognized TransferPolicyType: %v\n", val)
		}
	case "ArgScansAttempted":

	case "ArgScansCompleted":

	case "BWMeasurement":
		i, err := strconv.Atoi(val)
		if err != nil {
			parseErr = err
			break
		}
		sess.stats.MeasuredLinkRateKbps = i
	case "BlockInfo":

	case "Bytescont":
		i, err := strconv.ParseInt(val, 0, 64)
		if err != nil {
			parseErr = err
			break
		}
		tempFileStats.ContiguousBytes = i
	case "ChunkSize":
		i, err := strconv.Atoi(val)
		if err != nil {
			parseErr = err
			break
		}
		sess.chunkSize = i
	case "Code":
		i, err := strconv.Atoi(val)
		if err != nil {
			parseErr = err
			break
		}
		sess.stats.ErrorCode = i
		tempFileStats.ErrorCode = i
	case "Cookie":
		sess.stats.Cookie = val
	case "Delay":
		i, err := strconv.Atoi(val)
		if err != nil {
			parseErr = err
			break
		}
		sess.stats.DelayMs = i
	case "Description":
		sess.stats.ErrorDescription = val
		tempFileStats.ErrorDescription = val
	case "Destination":
		sess.stats.DestPath = val
	case "Direction":
		sess.stats.Direction = val
	case "DiskInfo":

	case "Elapsedusec":
		i, err := strconv.Atoi(val)
		if err != nil {
			parseErr = err
			break
		}
		sess.stats.ElapsedUSec = i
	case "Encryption":
		if val == "Yes" {
			sess.stats.Encryption = true
		} else {
			sess.stats.Encryption = false
		}
	case "EndByte":
		i, err := strconv.ParseInt(val, 0, 64)
		if err != nil {
			parseErr = err
			break
		}
		tempFileStats.EndByte = i
	case "FileBytes":
		i, err := strconv.ParseUint(val, 0, 64)
		if err != nil {
			parseErr = err
			break
		}
		sess.stats.TotalWrittenBytes = i
	case "File":
		tempFileStats.Name = val
	case "FileScansCompleted":

	case "Host":
		sess.stats.Host = val
	case "Loss":
		i, err := strconv.ParseUint(val, 0, 64)
		if err != nil {
			parseErr = err
			break
		}
		sess.stats.TotalLostBytes = i
	case "ManifestFile":
		sess.stats.ManifestFilePath = val
	case "MinRate":
		i, err := strconv.Atoi(val)
		if err != nil {
			parseErr = err
			break
		}
		sess.stats.MinRateKbps = i
	case "Operation":

	case "Password":
		// sess.Password = val
	case "PathScansAttempted":
		i, err := strconv.Atoi(val)
		if err != nil {
			parseErr = err
			break
		}
		sess.stats.SourcePathsScanAttempted = i
	case "PathScansExcluded":
		i, err := strconv.Atoi(val)
		if err != nil {
			parseErr = err
			break
		}
		sess.stats.SourcePathsScanExcluded = i
	case "PathScansFailed":
		i, err := strconv.Atoi(val)
		if err != nil {
			parseErr = err
			break
		}
		sess.stats.SourcePathsScanFailed = i
	case "PathScansIrregular":
		i, err := strconv.Atoi(val)
		if err != nil {
			parseErr = err
			break
		}
		sess.stats.SourcePathsScanIrregular = i
	case "PMTU":
		i, err := strconv.Atoi(val)
		if err != nil {
			parseErr = err
			break
		}
		sess.stats.Pmtu = i
	case "Policy":
		// See key "Adaptive"
	case "Port":
		i, err := strconv.Atoi(val)
		if err != nil {
			parseErr = err
			break
		}
		sess.stats.UdpPort = i
	case "PreTransferBytes":
		i, err := strconv.Atoi(val)
		if err != nil {
			parseErr = err
			break
		}
		sess.stats.PreCalcTotalBytes = i
	case "PreTransferDirs":

	case "PreTransferSpecial":

	case "PreTransferFiles":
		i, err := strconv.Atoi(val)
		if err != nil {
			parseErr = err
			break
		}
		sess.stats.PreCalcTotalFiles = i
	case "Priority":
		if val == "1" {
			sess.stats.Policy = HIGH_TRANSFER_POLICY
		}
	case "Progress":

	case "Query":

	case "QueryResponse":

	case "Rate":
		i, err := strconv.Atoi(val)
		if err != nil {
			parseErr = err
			break
		}
		sess.stats.TargetRateKbps = i
	case "Remote":
		if val == "Yes" {
			sess.isRemote = true
			sess.stats.Remote = true
		} else {
			sess.isRemote = false
			sess.stats.Remote = false
		}
	case "ServiceLevel":
		i, err := strconv.Atoi(val)
		if err != nil {
			parseErr = err
			break
		}
		sess.stats.BandwidthCapKbps = i
	case "SessionId":
		sess.stats.SessionId = val
	case "Size":
		i, err := strconv.ParseInt(val, 0, 64)
		if err != nil {
			parseErr = err
			break
		}
		tempFileStats.SizeBytes = i
	case "Source":
		sess.stats.SourcePaths = val
	case "StartByte":
		i, err := strconv.ParseInt(val, 0, 64)
		if err != nil {
			parseErr = err
			break
		}
		tempFileStats.StartByte = i
	case "Token":
		sess.stats.Token = val
	case "TransferBytes":
		i, err := strconv.ParseUint(val, 0, 64)
		if err != nil {
			parseErr = err
			break
		}
		sess.stats.TotalTransferredBytes = i
	case "TransfersAttempted":
		i, err := strconv.Atoi(val)
		if err != nil {
			parseErr = err
			break
		}
		sess.stats.TransfersAttempted = i
	case "TransfersFailed":
		i, err := strconv.Atoi(val)
		if err != nil {
			parseErr = err
			break
		}
		sess.stats.TransfersFailed = i
	case "TransfersPassed":
		i, err := strconv.Atoi(val)
		if err != nil {
			parseErr = err
			break
		}
		sess.stats.TransfersPassed = i
	case "TransfersSkipped":
		i, err := strconv.Atoi(val)
		if err != nil {
			parseErr = err
			break
		}
		sess.stats.TransfersSkipped = i
	case "Type":
		switch val {
		case "DONE":
			sess.stats.State = FINISHED_SESSION_STATE
		case "ERROR":
			sess.stats.State = FAILED_SESSION_STATE
		case "INIT":
			sess.stats.State = CONNECTING_SESSION_STATE
		case "QUERY":
			sess.stats.State = AUTHENTICATING_SESSION_STATE
		case "SESSION":
			sess.stats.State = STARTED_SESSION_STATE
		case "STOP":
			sess.stats.FilesComplete += 1
			sess.stats.State = TRANSFERRING_SESSION_STATE
			tempFileStats.State = FINISHED_FILE_STATE
		case "SKIP":
			sess.stats.FilesComplete += 1
			sess.stats.FilesSkipped += 1
			sess.stats.State = TRANSFERRING_SESSION_STATE
			tempFileStats.State = SKIPPED_FILE_STATE
		case "FILEERROR":
			sess.stats.FilesFailed += 1
			sess.stats.State = TRANSFERRING_SESSION_STATE
			tempFileStats.State = FAILED_FILE_STATE
		case "STATS":
			sess.stats.State = TRANSFERRING_SESSION_STATE
			tempFileStats.State = TRANSFERRING_FILE_STATE
		case "START", "NOTIFICATION":
			sess.stats.State = TRANSFERRING_SESSION_STATE
		default:
			return errors.New(val + " not recognized as management message type.")
		}
	case "User":
		sess.stats.User = val
	case "UserStr":

	case "Written":
		i, err := strconv.ParseInt(val, 0, 64)
		if err != nil {
			parseErr = err
			break
		}
		tempFileStats.WrittenBytes = i
	case "XferId":

	case "XferRetry":

	case "Tags":

	case "TCPPort":
		i, err := strconv.Atoi(val)
		if err != nil {
			parseErr = err
			break
		}
		sess.stats.TcpPort = i
	case "ServerNodeId":
		sess.stats.ServerNodeId = val
	case "ServerClusterId":
		sess.stats.ServerClusterId = val
	case "ClientNodeId":
		sess.stats.ClientNodeId = val
	case "ClientClusterId":
		sess.stats.ClientClusterId = val
	case "FileChecksumType":
		tempFileStats.FileChecksumType = val
	case "FileChecksum":
		tempFileStats.FileChecksum = val
	default:
	}

	if parseErr != nil {
		return fmt.Errorf("TransferSession: updateSessionStats() could not parse "+val+" as "+key+": %v",
			parseErr)
	}

	return nil
}
