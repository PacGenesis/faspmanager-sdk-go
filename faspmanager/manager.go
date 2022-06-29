package faspmanager

import (
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

	"github.com/twinj/uuid"
)

// Manager is responsible for monitoring FASP transfers started locally through a
// call to Manager.StartTransfer and remotely if Manager.ListenForRemoteSessions is
// set.
type Manager struct {
	// The location of the ascp executable. If it cannot be found in the environment's
	// PATH variable, it must be explicitly set prior to starting any transfers.
	ascpPath string

	// If true, the manager will remove a TransferSession from record when it completes.
	// Any attempted action which targets a completed session will return an ErrTransferDNE.
	removeCompletedSessions bool

	// The port-number of the TCP listener which listens for management connections.
	managementPort int

	// Dictates whether the management port is published, and thus whether management
	// of externally initiated sessions is enabled.
	isPublishingMgmtPort bool

	// Location of the file publishing the management port, if it exists.
	managementPortFilePath string

	// Map of unique session ID's to session pointers.
	sessions     map[SessionId]*transferSession
	sessionsSync sync.RWMutex

	// Global listeners - these are notified of every management message received.
	globalListeners []TransferListener
	listenersSync   sync.RWMutex
	globalListener  globalListenerWrapper

	// Channel to send termination signal to worker goroutines.
	terminate chan struct{}
}

var (
	ErrNoAscp     = errors.New("Manager: No ASCP executable found. Be sure the location of the ASCP executable is on your path.")
	ErrSessionDNE = errors.New("Manager: No TransferSession exists with the given ID. If you have set " +
		"Manager.RemoveCompletedSessions to true, it may have been removed from record.")
)

// Satisfies the TransferListener interface so that the manager can pass
// TransferEvents to global listeners without itself implementing the interface.
type globalListenerWrapper struct {
	m *Manager
}

func (g globalListenerWrapper) OnTransferEvent(e TransferEvent) {
	g.m.listenersSync.Lock()
	for _, listener := range g.m.globalListeners {
		listener.OnTransferEvent(e)
	}
	g.m.listenersSync.Unlock()
}

// Used to set the timeout on network actions. As used here, timeout does not
// signify that a network action has been unsuccessful, but rather that a check
// should be performed to see if the manager has sent a termination signal, in
// which case network resources should be cleaned up and closed.
var workerCloseDeadline = time.Second

// New creates a new Manager listening on an open port. Returns an error if the
// manager is unable to create a TCPListener to receive transfer updates.
func New() (*Manager, error) {
	return NewWithPort(0)
}

// NewWithPort creates a new Manager listening on the given port. Returns an error
// if the manager is unable to create a TCPListener to receive transfer updates or
// if the port is not in the range [0, 65535]. NewWithPort(0) creates a manager
// which listens on an OS-assigned port.
func NewWithPort(port int) (*Manager, error) {
	if port < 0 || port > 65535 {
		return nil, errors.New("Manager: Port must be in range [0, 65535]")
	}

	addr, err := net.ResolveTCPAddr("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		return nil, fmt.Errorf("Manager: Unable to resolve TCP address for monitoring transfers. Error: %v.", err)
	}

	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("Manager: Unable to create TCPListener for monitoring transfers. Error %v.", err)
	}
	fmLog.Println("Manager: Listening on port ", listener.Addr().(*net.TCPAddr).Port)

	manager := &Manager{
		managementPort:          listener.Addr().(*net.TCPAddr).Port,
		terminate:               make(chan struct{}),
		sessions:                make(map[SessionId]*transferSession),
		sessionsSync:            sync.RWMutex{},
		listenersSync:           sync.RWMutex{},
		removeCompletedSessions: false,
	}
	manager.globalListener = globalListenerWrapper{manager}

	go manager.listenForManagementConns(listener)

	return manager, nil
}

func (m *Manager) listenForManagementConns(listener *net.TCPListener) {
	defer listener.Close()
	for {
		select {
		case <-m.terminate:
			fmLog.Println("Manager: Management TCP Listener terminating.")
			return
		default:
		}
		listener.SetDeadline(time.Now().Add(workerCloseDeadline))
		conn, err := listener.AcceptTCP()
		if err != nil {
			// Either the listener has timed out or there has been a network error.
			continue
		}
		fmLog.Println("Manager: Management connection opened.")
		go m.handleNewManagementConn(conn)
	}
}

func (m *Manager) handleNewManagementConn(conn *net.TCPConn) {

	// Find out which TransferSession the connection belongs to by looking for
	// the "UserStr" field in the first management message.
	managementMessageBuffer := make([]byte, 2048)
	bytesReadSoFar := 0
	for {
		select {
		case <-m.terminate:
			conn.Close()
			return
		default:
		}
		conn.SetReadDeadline(time.Now().Add(workerCloseDeadline))
		numBytesRead, err := conn.Read(managementMessageBuffer[bytesReadSoFar:])
		if err != nil {
			if err, ok := err.(net.Error); !ok || !err.Timeout() {
				// Error is not a timeout error.
				conn.Close()
				return
			}
		}
		bytesReadSoFar += numBytesRead

		message := string(managementMessageBuffer[:bytesReadSoFar])
		sessId, notFoundErr := getSessionIdFromMessage(message)
		if notFoundErr != nil {
			// The segment of the message we've received so far doesn't contain a session ID.
			continue
		}

		sess, exists := m.concSessionsGet(sessId)
		if !exists {

			// The transfer is remotely initiated. Create a new session for it.
			sess = &transferSession{
				id:            SessionId(generateUniqueId()),
				stats:         &SessionStatistics{},
				statsSync:     sync.RWMutex{},
				listeners:     []TransferListener{m.globalListener},
				listenersSync: sync.Mutex{},
				fileStats:     make(map[string]*FileStatistics),
				fileStatsSync: sync.Mutex{},
				started:       make(chan struct{}),
				complete:      make(chan struct{}),
				terminate:     m.terminate,
				onComplete:    func(id SessionId) { m.onSessionComplete(id) },
			}
			m.concSessionsAdd(sess.id, sess)
		}

		// Pass the first management message to the session.
		err = sess.receiveManagementBlob(message)
		if err != nil {
			fmLog.Printf("Manager: Error processing management message: %v\nMessage:\n%v\n", err, message)
		}

		// Pass the connection off to its session.
		err = sess.setManagementConn(conn)
		if err != nil {
			fmLog.Printf("Manager: Error setting session's management connection: %v\n", err)
		}

		return
	}
}

func getSessionIdFromMessage(managementMsg string) (SessionId, error) {

	// The SessionId is the 'UserStr' value in the message.
	lines := strings.Split(managementMsg, "\n")
	for _, line := range lines {
		if strings.Contains(line, "UserStr: ") {
			return SessionId(line[len("UserStr: "):]), nil
		}
	}
	return SessionId(""), errors.New("Manager: Could not find 'UserStr' field in management message: " + managementMsg)
}

// StartTransfer starts a transfer session based on the given order, which may
// be a FileDownload, FileUpload, PersistentDownload, // PersistentUpload,
// MultiFileUpload, or MultiFileDownload. Returns the session's SessionId, or an
// error indicating failure.
func (m *Manager) StartTransfer(order TransferOrder) (SessionId, error) {
	return m.StartTransferWithListener(order, nil)
}

// StartTransferWithListener starts a transfer session. The given listener will be
// notified of the session's TransferEvents. Be aware that the listener will be
// called from a separate goroutine as events come in.  Resources should be managed
// accordingly.
func (m *Manager) StartTransferWithListener(order TransferOrder, l TransferListener) (SessionId, error) {
	ids, err := m.StartMultiProcessTransferWithListener(order, 1, l)
	if err != nil {
		return SessionId(""), err
	}
	return ids[0], nil
}

// StartMultiProcessTransfer starts a multi-process transfer session. Each process
// will have a unique SessionId.  Returns a slice containing these id's. Note that
// a PersistentUpload or PersistentDownload order cannot be used to start a
// multi-process transfer.
func (m *Manager) StartMultiProcessTransfer(order TransferOrder, numProcesses int) ([]SessionId, error) {
	return m.StartMultiProcessTransferWithListener(order, numProcesses, nil)
}

// StartMultiProcessTransferWithListener starts a multi-process transfer session
// with the given TransferListener.  Returns a slice of the SessionId's
// corresponding to each process. The given listener will be subscribed to each of
// the processes' TransferEvents and will be invoked on a separate goroutine as
// events come in.  Note that a PersistentUpload or PersistentDownload cannot be
// used to start a multi-process transfer.
func (m *Manager) StartMultiProcessTransferWithListener(order TransferOrder, numProcesses int, l TransferListener) ([]SessionId, error) {
	ascpPath, notFoundErr := m.getOrFindAscpPath()
	if notFoundErr != nil {
		return nil, notFoundErr
	}
	ascpOpts, err := order.GetAscpOptions()
	if err != nil {
		return nil, err
	}
	if ascpOpts.TransferOpts == nil {
		ascpOpts.TransferOpts = &TransferOptions{}
	}
	if numProcesses < 1 {
		return nil, errors.New("Manager: numProcesses must be positive. Was: " + strconv.Itoa(numProcesses))
	}
	if numProcesses > 1 && ascpOpts.Persistent {
		return nil, errors.New("Manager: Cannot perform multiprocess transfer with a persistent session.")
	}

	var sessionIds []SessionId
	for i := 1; i <= numProcesses; i++ {
		sessionId := SessionId(generateUniqueId())
		sessionOpts := sessionOptions{
			ascpPath:       ascpPath,
			id:             sessionId,
			processes:      numProcesses,
			processIndex:   i,
			managementPort: m.managementPort,
			terminate:      m.terminate,
			onComplete:     func(id SessionId) { m.onSessionComplete(id) },
		}
		session, err := newTransferSession(sessionOpts, ascpOpts)
		if err != nil {
			return nil, err
		}
		session.addListener(m.globalListener)
		if l != nil {
			session.addListener(l)
		}
		err = session.start()
		if err != nil {
			return nil, err
		}
		m.concSessionsAdd(sessionId, session)
		sessionIds = append(sessionIds, sessionId)
	}
	return sessionIds, nil
}

// onSessionComplete is invoked on a worker goroutine when a session completes.
func (m *Manager) onSessionComplete(id SessionId) {
	if m.removeCompletedSessions {
		m.concSessionsRemove(id)
	}
}

// AddFileSource adds a source to a persistent session, which begins a transfer
// from sourcePath to destPath. Returns an error if the targeted session is not
// found or is not persistent.
func (m *Manager) AddFileSource(id SessionId, sourcePath, destPath string) error {
	return m.AddFileSourceRange(id, sourcePath, destPath, 0, 0)
}

// AddFileSourceRange adds a source to a persistent transfer session, which begins
// a transfer from sourcePath to destPath of the range of bytes beginning at offset
// startByte and ending at offset endByte in the source file.  Returns an error if
// the targeted session is not found or is not persistent. A call of this method
// with startByte = 0 and endByte = 0 is equivalent to a call to
// Manager.AddFileSource.
func (m *Manager) AddFileSourceRange(id SessionId, sourcePath string, destPath string, startByte, endByte int64) error {
	sess, exists := m.concSessionsGet(id)
	if !exists {
		return ErrSessionDNE
	}
	return sess.addFileSource(sourcePath, destPath, startByte, endByte)
}

// AddReaderSource adds a Reader source to a persistent stream upload.  Bytes read
// by the Reader will be sent to the remote destination until a call to reader.Read
// returns io.EOF (signalling healthy completion) or another error.
//
// Returns an error if no session is found with the given id, if the targeted
// session is not a persistent stream upload. Listeners are notified of a
// faspmanager.FILE_ERROR_TRANSFER_EVENT if a network error occurs writing to ascp
// management, or a non-io.EOF error occurs while reading from the reader.
// Terminating the session by calling Manager.Close will stop the transfer prior to
// the next read.  This functionality requires ascp4 and does not block.
func (m *Manager) AddReaderSource(id SessionId, reader io.Reader, destPath string) error {
	sess, exists := m.concSessionsGet(id)
	if !exists {
		return ErrSessionDNE
	}
	return sess.addReaderSource(reader, destPath)
}

// AddStreamReaderSource adds a StreamReader source to a persistent stream
// upload. As bytes are read from the StreamReader, they will be transferred to
// the session's  remote host and written to the given destination path at the
// offset specified by  StreamerReader.Read's StreamBlock.Position value.
//
// Returns an error if no session is found with the given id, or if the
// target session is not a persistent stream upload.  Listeners are notified of
// a faspmanager.FILE_ERROR_TRANSFER_EVENT if a network error occurs writing to
// ascp management, or a non-io.EOF error occurs while reading from the reader.
// Terminating the session by calling Manager.Close will stop the transfer prior
// to the next read. This functionality requires ascp4 and does not block.
func (m *Manager) AddStreamReaderSource(id SessionId, reader StreamReader, destPath string) error {
	sess, exists := m.concSessionsGet(id)
	if !exists {
		return ErrSessionDNE
	}
	return sess.addStreamReaderSource(reader, destPath)
}

// AddGlobalListener adds a global TransferListener. The listener will be
// notified of all TransferEvents for all ongoing transfer sessions.
func (m *Manager) AddGlobalListener(l TransferListener) {
	m.listenersSync.Lock()
	m.globalListeners = append(m.globalListeners, l)
	m.listenersSync.Unlock()
}

// AddSessionListener adds a session-specific TransferListener. The listener
// will only be notified of TransferEvents pertaining to the session with the
// given id. Returns an error if no session exists with the given id, or if the
// session is complete and Manager.RemoveCompletedSessions is true.
func (m *Manager) AddSessionListener(id SessionId, l TransferListener) error {
	sess, exists := m.concSessionsGet(id)
	if !exists {
		return ErrSessionDNE
	}
	sess.addListener(l)
	return nil
}

// RemoveGlobalListener removes a global TransferListener. Returns true if the
// listener was removed, false if it was not found.
func (m *Manager) RemoveGlobalListener(l TransferListener) bool {
	m.listenersSync.Lock()
	defer m.listenersSync.Unlock()
	for i, listener := range m.globalListeners {
		if listener == l {
			m.globalListeners = append(m.globalListeners[:i], m.globalListeners[i+1:]...)
			return true
		}
	}
	return false
}

// RemoveSessionListener removes a session-specific TransferListener. Returns
// true if the listener was removed, false if not, and an error if no session
// exists with the given id.
func (m *Manager) RemoveSessionListener(id SessionId, l TransferListener) (bool, error) {
	sess, exists := m.concSessionsGet(id)
	if !exists {
		return false, ErrSessionDNE
	}
	return sess.removeListener(l), nil
}

// ListenForRemoteSessions dictates whether the manager should listen for
// remotely initiated sessions. If set, the manager can manage transfers
// initiated outside itself and any global listener will be notified of events
// on such transfers as well. Default: Off.
func (m *Manager) ListenForRemoteSessions(on bool) error {
	ascpPath, notFoundErr := m.getOrFindAscpPath()
	if notFoundErr != nil {
		return notFoundErr
	}
	if on && !m.isPublishingMgmtPort {
		portFilePath, err := getPortFilePath(ascpPath)
		if err != nil {
			return err
		}
		err = writePortFile(portFilePath, m.managementPort)
		if err != nil {
			return fmt.Errorf("Manager: Cannot publish management port. Creation of port file returned error: %v", err)
		}
		m.managementPortFilePath = portFilePath
		m.isPublishingMgmtPort = true
	} else if !on && m.isPublishingMgmtPort {
		err := os.Remove(m.managementPortFilePath)
		if err != nil {
			return fmt.Errorf("Manager: Cannot unpublish management port. Removal of port file returned error: %v", err)
		}
		m.managementPortFilePath = ""
		m.isPublishingMgmtPort = false
	}
	return nil
}

// IsListeningForRemoteSessions returns whether the manager is listening for
// remotely initiated sessinos.
func (m *Manager) IsListeningForRemoteSessions() bool {
	return m.isPublishingMgmtPort
}

// RemoteCompletedSessions dictates whether the manager maintains record of
// transfer sessions after their completion. If true, the manager will remove a
// transfer session from record when it completes, and any attempted action
// which targets a completed session will return an ErrTransferDNE.
func (m *Manager) RemoveCompletedSessions(on bool) {
	m.removeCompletedSessions = on
}

// IsRemovingCompletedSessions returns whether the manager is removing record of transfer sessions after their completion.
// Control this option with Manager.RemoveCompletedSessions.
func (m *Manager) IsRemovingCompletedSessions() bool {
	return m.removeCompletedSessions
}

// GetSessionStats returns the SessionStatistics of the session with the given
// id, or an error if the session is not found.
func (m *Manager) GetSessionStats(id SessionId) (*SessionStatistics, error) {
	sess, exists := m.concSessionsGet(id)
	if !exists {
		return nil, ErrSessionDNE
	}
	return sess.getStatsCopy(), nil
}

// SetRate sets the transfer rate of the session with the given id. Returns an
// error if the session is not found or if the session has not started or is
// complete.
func (m *Manager) SetRate(id SessionId, targetRateKbps int, minRateKbps int,
	policy TransferPolicyType) error {
	session, exists := m.concSessionsGet(id)
	if !exists {
		return ErrSessionDNE
	}
	return session.setRate(targetRateKbps, minRateKbps, policy)
}

// LockPersistentSession locks a persistent session. A locked persistent session
// cannot have sources added and will be complete after any of its remaining
// transfers finish. Returns an error if no session is found with the given id
// or if the session is not persistent or is not ongoing.
func (m *Manager) LockPersistentSession(id SessionId) error {
	sess, exists := m.concSessionsGet(id)
	if !exists {
		return ErrSessionDNE
	}
	return sess.lock()
}

// CancelSession cancels the session with the given id. Returns an error if the
// session is not found, is in a state where it cannot be cancelled, or if the
// attempt to cancel the session fails due to a network error.
func (m *Manager) CancelSession(id SessionId) error {
	sess, exists := m.concSessionsGet(id)
	if !exists {
		return ErrSessionDNE
	}
	return sess.cancel()
}

// WaitOnSessionStart blocks until the session with the given ID begins
// receiving transfer updates. Returns an error if the session is not found or
// if the transfer process exits unexpectedly.
func (m *Manager) WaitOnSessionStart(id SessionId) error {
	sess, exists := m.concSessionsGet(id)
	if !exists {
		return ErrSessionDNE
	}
	return sess.waitOnStart()
}

// WaitOnSessionStop blocks until the transfer session with the given id stops.
// Returns an error if the session is not found or if the transfer process exits
// unexpectedly.
func (m *Manager) WaitOnSessionStop(id SessionId) error {
	sess, exists := m.concSessionsGet(id)
	if !exists {
		return ErrSessionDNE
	}
	return sess.waitOnStop()
}

// GetSessionIds returns the id's of all transfer sessions tracked by the manager.
func (m *Manager) GetSessionIds() (ids []SessionId) {
	m.sessionsSync.RLock()
	for id, _ := range m.sessions {
		ids = append(ids, id)
	}
	m.sessionsSync.RUnlock()
	return
}

// SetAscpPath points the manager to the location of the ASCP executable. If the
// ASCP executable is not found in the environment's PATH, its location must be
// set here prior to starting any transfers.
func (m *Manager) SetAscpPath(path string) error {
	path, err := exec.LookPath(path)
	if err != nil {
		return fmt.Errorf("Manager: Given ASCP filepath does not exist: %v\n", err)
	}
	m.ascpPath = path
	return nil
}

// Close closes the Manager, cancelling any ongoing TransferSessions. Any
// attached listeners will no longer be notified of TransferEvents, and
// information pertaining to completed or ongoing transfers will be lost. This
// should only be called when use of the manager is complete.
func (m *Manager) Close() {
	m.sessionsSync.Lock()
	for _, sess := range m.sessions {
		if !sess.isComplete() && !sess.isRemote {
			// This may return an error, but we ignore it here.
			sess.cancel()
		}
	}
	m.sessions = make(map[SessionId]*transferSession)
	m.sessionsSync.Unlock()
	close(m.terminate)
	// Do not wait for worker goroutines and network connections to terminate.
}

func (m *Manager) concSessionsAdd(id SessionId, sess *transferSession) {
	m.sessionsSync.Lock()
	m.sessions[id] = sess
	m.sessionsSync.Unlock()
}

func (m *Manager) concSessionsGet(id SessionId) (*transferSession, bool) {
	m.sessionsSync.RLock()
	sess, exists := m.sessions[id]
	m.sessionsSync.RUnlock()
	return sess, exists
}

func (m *Manager) concSessionsRemove(id SessionId) {
	m.sessionsSync.Lock()
	delete(m.sessions, id)
	m.sessionsSync.Unlock()
}

func (m *Manager) getOrFindAscpPath() (string, error) {
	if len(m.ascpPath) == 0 {
		if ascpPath, err := exec.LookPath("ascp"); err != nil {
			return "", ErrNoAscp
		} else {
			m.ascpPath = ascpPath
		}
	}
	return m.ascpPath, nil
}

func generateUniqueId() string {
	return uuid.NewV4().String()
}
