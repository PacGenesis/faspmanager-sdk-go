package faspmanager

import (
	"net"
	"strconv"
	"testing"
	"time"
)

// Implements the net.Conn interface.
type mockMgmtConn struct {
	lastSentMsg string
}

func (m *mockMgmtConn) Write(b []byte) (n int, err error) {
	m.lastSentMsg = string(b)
	return len(b), nil
}

func (m *mockMgmtConn) Read(b []byte) (n int, err error)   { return 0, nil }
func (m *mockMgmtConn) Close() error                       { return nil }
func (m *mockMgmtConn) LocalAddr() net.Addr                { return nil }
func (m *mockMgmtConn) RemoteAddr() net.Addr               { return nil }
func (m *mockMgmtConn) SetDeadline(t time.Time) error      { return nil }
func (m *mockMgmtConn) SetReadDeadline(t time.Time) error  { return nil }
func (m *mockMgmtConn) SetWriteDeadline(t time.Time) error { return nil }

// Implements the faspmanager.TransferListener interface.
type testTransferEventListener struct {
	lastEvent TransferEvent
}

func (l *testTransferEventListener) OnTransferEvent(e TransferEvent) {
	l.lastEvent = e
}

const testSessionId = "testId"

var testSessionOpts = sessionOptions{
	ascpPath:       "ascp",
	id:             SessionId(testSessionId),
	processes:      1,
	processIndex:   1,
	managementPort: 33001,
	terminate:      nil,
	onComplete:     nil,
}

var testAscpOpts_NonPersistent = &AscpOptions{
	SourcePaths: []string{"sourcepath.txt"},
	DestPaths:   []string{"destpaths.txt"},
	RemoteHost:  "host",
	RemoteUser:  "user",
	RemotePass:  "pass",
	Persistent:  false,
	Direction:   UploadDirection,
}

var testAscpOpts_Persistent = &AscpOptions{
	SourcePaths: nil,
	DestPaths:   nil,
	RemoteHost:  "host",
	RemoteUser:  "user",
	RemotePass:  "pass",
	Persistent:  true,
	Direction:   UploadDirection,
}

func TestNewSession_InitializesStats(t *testing.T) {
	transferOpts := &TransferOptions{
		DisableEncryption: false,
		Policy:            FAIR_TRANSFER_POLICY,
		Token:             "token",
		MinRateKbps:       2000,
		TargetRateKbps:    3000,
		TransferRetry:     3,
		UdpPort:           60000,
		TcpPort:           50000,
	}

	ascpOpts := &AscpOptions{
		SourcePaths:  []string{"source.txt"},
		DestPaths:    []string{"/destination.txt"},
		RemoteUser:   "user",
		RemoteHost:   "host",
		RemotePass:   "pass",
		Persistent:   false,
		Direction:    UploadDirection,
		TransferOpts: transferOpts,
	}

	sessionOpts := sessionOptions{
		id: SessionId("id"),
	}

	sess, err := newTransferSession(sessionOpts, ascpOpts)
	if err != nil {
		t.Fatal(err)
	}

	stats := sess.getStatsCopy()

	if stats.Encryption != !transferOpts.DisableEncryption {
		t.Fatalf("Incorrect encryption. Got %v. Expected %v.", stats.Encryption, !transferOpts.DisableEncryption)
	}
	if stats.Policy != transferOpts.Policy {
		t.Fatalf("Incorrect policy. Got %v. Expected %v.", stats.Policy, transferOpts.Policy)
	}
	if stats.Token != transferOpts.Token {
		t.Fatalf("Incorrect token. Got %v. Expected %v.", stats.Token, transferOpts.Token)
	}
	if stats.MinRateKbps != transferOpts.MinRateKbps {
		t.Fatalf("Incorrect min rate. Got %v. Expected %v.", stats.MinRateKbps, transferOpts.MinRateKbps)
	}
	if stats.TargetRateKbps != transferOpts.TargetRateKbps {
		t.Fatalf("Incorrect target rate. Got %v. Expected %v.", stats.TargetRateKbps, transferOpts.TargetRateKbps)
	}
	if stats.TransferRetry != transferOpts.TransferRetry {
		t.Fatalf("Incorrect transfer retry. Got %v. Expected %v.", stats.TransferRetry, transferOpts.TransferRetry)
	}
	if stats.UdpPort != transferOpts.UdpPort {
		t.Fatalf("Incorrect UDP port. Got %v. Expected %v.", stats.UdpPort, transferOpts.UdpPort)
	}
	if stats.TcpPort != transferOpts.TcpPort {
		t.Fatalf("Incorrect TCP port. Got %v. Expected %v.", stats.TcpPort, transferOpts.TcpPort)
	}
	if stats.Id != sessionOpts.id {
		t.Fatalf("Incorrect id. Got %v. Expected %v.", stats.Id, sessionOpts.id)
	}
}

func TestAddSource(t *testing.T) {
	sess, err := newTransferSession(testSessionOpts, testAscpOpts_Persistent)
	if err != nil {
		t.Fatal(err)
	}

	mockConn := &mockMgmtConn{}
	err = sess.setManagementConn(mockConn)
	if err != nil {
		t.Fatal(err)
	}

	sourcePath := "sourcepath.txt"
	destPath := "destpath.txt"

	sess.addFileSource(sourcePath, destPath, 0, 0)

	correctMsg := "FASPMGR 2\n"
	correctMsg += "Type: START\n"
	correctMsg += "Source: " + sourcePath + "\n"
	correctMsg += "Destination: " + destPath + "\n"
	correctMsg += "\n"

	if mockConn.lastSentMsg != correctMsg {
		t.Fatalf("Incorrect management message. Expected %s. Got %s", correctMsg, mockConn.lastSentMsg)
	}
}

func TestAddSource_ByteRange(t *testing.T) {
	sess, err := newTransferSession(testSessionOpts, testAscpOpts_Persistent)
	if err != nil {
		t.Fatal(err)
	}

	mockConn := &mockMgmtConn{}
	err = sess.setManagementConn(mockConn)
	if err != nil {
		t.Fatal(err)
	}

	sourcePath := "sourcepath.txt"
	destPath := "destpath.txt"
	var rangeStart int64 = 10
	var rangeEnd int64 = 100

	sess.addFileSource(sourcePath, destPath, rangeStart, rangeEnd)

	correctMsg := "FASPMGR 2\n"
	correctMsg += "Type: START\n"
	correctMsg += "Source: " + sourcePath + "\n"
	correctMsg += "Destination: " + destPath + "\n"
	correctMsg += "FileBytes: " + strconv.Itoa(int(rangeStart)) + ":" + strconv.Itoa(int(rangeEnd)) + "\n"
	correctMsg += "\n"

	if mockConn.lastSentMsg != correctMsg {
		t.Fatalf("Incorrect management message. Expected %s. Got %s", correctMsg, mockConn.lastSentMsg)
	}
}

func TestAddSource_NonPersistentSession(t *testing.T) {
	sess, err := newTransferSession(testSessionOpts, testAscpOpts_NonPersistent)
	if err != nil {
		t.Fatal(err)
	}

	err = sess.setManagementConn(&mockMgmtConn{})
	if err != nil {
		t.Fatal(err)
	}

	err = sess.addFileSource("", "", 0, 0)
	if err == nil {
		t.Fatal("addSource should return error when session is not persistent.")
	}
}

func TestAddSource_MgmtConnNotSet(t *testing.T) {
	sess, err := newTransferSession(testSessionOpts, testAscpOpts_Persistent)
	if err != nil {
		t.Fatal(err)
	}

	err = sess.addFileSource("", "", 0, 0)
	if err == nil {
		t.Fatal("addSource should return error when management connection is not set.")
	}

}

func TestAddSource_LockedSession(t *testing.T) {
	sess, err := newTransferSession(testSessionOpts, testAscpOpts_Persistent)
	if err != nil {
		t.Fatal(err)
	}

	err = sess.setManagementConn(&mockMgmtConn{})
	if err != nil {
		t.Fatal(err)
	}

	err = sess.lock()
	if err != nil {
		t.Fatal(err)
	}

	err = sess.addFileSource("", "", 0, 0)
	if err == nil {
		t.Fatal("addSource should return error when session is locked.")
	}
}

func TestAddSource_CompletedSession(t *testing.T) {
	sess, err := newTransferSession(testSessionOpts, testAscpOpts_Persistent)
	if err != nil {
		t.Fatal(err)
	}

	err = sess.setManagementConn(&mockMgmtConn{})
	if err != nil {
		t.Fatal(err)
	}

	close(sess.complete)

	err = sess.addFileSource("", "", 0, 0)
	if err == nil {
		t.Fatal("addSource should return error when session is completed.")
	}
}

func TestSetRate(t *testing.T) {
	sess, err := newTransferSession(testSessionOpts, testAscpOpts_Persistent)
	if err != nil {
		t.Fatal(err)
	}

	mockConn := &mockMgmtConn{}
	err = sess.setManagementConn(mockConn)
	if err != nil {
		t.Fatal(err)
	}

	targetRate := 20000
	minRate := 15000
	policy := FAIR_TRANSFER_POLICY

	err = sess.setRate(targetRate, minRate, policy)
	if err != nil {
		t.Fatal(err)
	}

	policyStr, err := policy.getManagementMessageString()
	if err != nil {
		t.Fatal(err)
	}

	correctMsg := "FASPMGR 2\n"
	correctMsg += "Type: RATE\n"
	correctMsg += "Rate: " + strconv.Itoa(targetRate) + "\n"
	correctMsg += "MinRate: " + strconv.Itoa(minRate) + "\n"
	correctMsg += "Adaptive: " + policyStr + "\n"
	correctMsg += "\n"

	if mockConn.lastSentMsg != correctMsg {
		t.Fatalf("Incorrect management message. Expected %s. Got %s", correctMsg, mockConn.lastSentMsg)
	}
}

func TestSetRate_MgmtConnNotSet(t *testing.T) {
	sess, err := newTransferSession(testSessionOpts, testAscpOpts_NonPersistent)
	if err != nil {
		t.Fatal(err)
	}

	err = sess.setRate(10000, 1000, FAIR_TRANSFER_POLICY)
	if err == nil {
		t.Fatal("setRate should return error when management connection is not set.")
	}
}

func TestSetRate_LockedSession(t *testing.T) {
	sess, err := newTransferSession(testSessionOpts, testAscpOpts_Persistent)
	if err != nil {
		t.Fatal(err)
	}

	err = sess.setManagementConn(&mockMgmtConn{})
	if err != nil {
		t.Fatal(err)
	}

	err = sess.lock()
	if err != nil {
		t.Fatal(err)
	}

	err = sess.setRate(10000, 1000, FAIR_TRANSFER_POLICY)
	if err == nil {
		t.Fatal("setRate should return error when session is locked.")
	}
}

func TestSetRate_CompletedSession(t *testing.T) {
	sess, err := newTransferSession(testSessionOpts, testAscpOpts_NonPersistent)
	if err != nil {
		t.Fatal(err)
	}

	err = sess.setManagementConn(&mockMgmtConn{})
	if err != nil {
		t.Fatal(err)
	}

	close(sess.complete)

	err = sess.setRate(10000, 1000, FAIR_TRANSFER_POLICY)
	if err == nil {
		t.Fatal("setRate should return error when session is complete.")
	}
}

func TestSetRate_BadPolicyString(t *testing.T) {
	sess, err := newTransferSession(testSessionOpts, testAscpOpts_NonPersistent)
	if err != nil {
		t.Fatal(err)
	}

	err = sess.setManagementConn(&mockMgmtConn{})
	if err != nil {
		t.Fatal(err)
	}

	err = sess.setRate(10000, 1000, TransferPolicyType(-1))
	if err == nil {
		t.Fatal("setRate should return error when transfer policy is invalid.")
	}
}

func TestLock(t *testing.T) {
	sess, err := newTransferSession(testSessionOpts, testAscpOpts_Persistent)
	if err != nil {
		t.Fatal(err)
	}

	mockConn := &mockMgmtConn{}
	err = sess.setManagementConn(mockConn)
	if err != nil {
		t.Fatal(err)
	}

	err = sess.lock()
	if err != nil {
		t.Fatal(err)
	}

	correctMsg := "FASPMGR 2\n"
	correctMsg += "Type: DONE\n"
	correctMsg += "Operation: Linger\n"
	correctMsg += "\n"

	if mockConn.lastSentMsg != correctMsg {
		t.Fatalf("Incorrect management message. Expected %s. Got %s", correctMsg, mockConn.lastSentMsg)
	}
}

func TestLock_NonPersistentSession(t *testing.T) {
	sess, err := newTransferSession(testSessionOpts, testAscpOpts_NonPersistent)
	if err != nil {
		t.Fatal(err)
	}

	mockConn := &mockMgmtConn{}
	err = sess.setManagementConn(mockConn)
	if err != nil {
		t.Fatal(err)
	}

	err = sess.lock()
	if err == nil {
		t.Fatal("lock should return error when session is not persistent.")
	}
}

func TestLock_MgmtConnNotSet(t *testing.T) {
	sess, err := newTransferSession(testSessionOpts, testAscpOpts_Persistent)
	if err != nil {
		t.Fatal(err)
	}

	err = sess.lock()
	if err == nil {
		t.Fatal("lock should return error when management connection is not set.")
	}
}

func TestLock_LockedSession(t *testing.T) {
	sess, err := newTransferSession(testSessionOpts, testAscpOpts_Persistent)
	if err != nil {
		t.Fatal(err)
	}

	mockConn := &mockMgmtConn{}
	err = sess.setManagementConn(mockConn)
	if err != nil {
		t.Fatal(err)
	}

	err = sess.lock()
	if err != nil {
		t.Fatal(err)
	}

	err = sess.lock()
	if err == nil {
		t.Fatal("lock should return error when session is already locked.")
	}
}

func TestLock_CompletedSession(t *testing.T) {
	sess, err := newTransferSession(testSessionOpts, testAscpOpts_Persistent)
	if err != nil {
		t.Fatal(err)
	}

	mockConn := &mockMgmtConn{}
	err = sess.setManagementConn(mockConn)
	if err != nil {
		t.Fatal(err)
	}

	close(sess.complete)

	err = sess.lock()
	if err == nil {
		t.Fatal("lock should return error when session is complete.")
	}
}

func TestCancel_PersistentSession(t *testing.T) {
	sess, err := newTransferSession(testSessionOpts, testAscpOpts_Persistent)
	if err != nil {
		t.Fatal(err)
	}

	mockConn := &mockMgmtConn{}
	err = sess.setManagementConn(mockConn)
	if err != nil {
		t.Fatal(err)
	}

	err = sess.cancel()
	if err != nil {
		t.Fatal(err)
	}

	correctMsg := "FASPMGR 2\n"
	correctMsg += "Type: DONE\n"
	correctMsg += "\n"

	if mockConn.lastSentMsg != correctMsg {
		t.Fatalf("Incorrect management message. Expected %s. Got %s", correctMsg, mockConn.lastSentMsg)
	}
}

func TestCancel_NonPersistentSession(t *testing.T) {
	sess, err := newTransferSession(testSessionOpts, testAscpOpts_NonPersistent)
	if err != nil {
		t.Fatal(err)
	}

	mockConn := &mockMgmtConn{}
	err = sess.setManagementConn(mockConn)
	if err != nil {
		t.Fatal(err)
	}

	err = sess.cancel()
	if err != nil {
		t.Fatal(err)
	}

	correctMsg := "FASPMGR 2\n"
	correctMsg += "Type: CANCEL\n"
	correctMsg += "\n"

	if mockConn.lastSentMsg != correctMsg {
		t.Fatalf("Incorrect management message. Expected %s. Got %s", correctMsg, mockConn.lastSentMsg)
	}
}

func TestCancel_MgmtConnNotSet(t *testing.T) {
	sess, err := newTransferSession(testSessionOpts, testAscpOpts_NonPersistent)
	if err != nil {
		t.Fatal(err)
	}

	err = sess.cancel()
	if err == nil {
		t.Fatal("cancel should return error when management connection is not set.")
	}
}

func TestCancel_LockedSession(t *testing.T) {
	sess, err := newTransferSession(testSessionOpts, testAscpOpts_Persistent)
	if err != nil {
		t.Fatal(err)
	}

	err = sess.setManagementConn(&mockMgmtConn{})
	if err != nil {
		t.Fatal(err)
	}

	err = sess.lock()
	if err != nil {
		t.Fatal(err)
	}

	err = sess.cancel()
	if err != nil {
		t.Fatal(err)
	}
}

func TestCancel_CompletedSession(t *testing.T) {
	sess, err := newTransferSession(testSessionOpts, testAscpOpts_Persistent)
	if err != nil {
		t.Fatal(err)
	}

	err = sess.setManagementConn(&mockMgmtConn{})
	if err != nil {
		t.Fatal(err)
	}

	close(sess.complete)

	err = sess.cancel()
	if err == nil {
		t.Fatal("cancel should return error when session is completed.")
	}
}

// --- Management Message Reception ------

func TestReceiveManagementMessage_Init(t *testing.T) {
	sess, err := newTransferSession(testSessionOpts, testAscpOpts_NonPersistent)
	if err != nil {
		t.Fatal(err)
	}

	l := &testTransferEventListener{}
	sess.addListener(l)

	msg := "FASPMGR 2\n"
	msg += "Type: INIT\n"
	msg += "SessionId: 123456789\n"
	msg += "Direction: Send\n"
	msg += "UserStr: " + testSessionId + "\n"
	msg += "Operation: Transfer\n"
	msg += "\n"

	err = sess.receiveManagementBlob(msg)
	if err != nil {
		t.Fatal(err)
	}

	if l.lastEvent == (TransferEvent{}) || l.lastEvent.SessionStats == nil {
		t.Fatal("listener did not receive TransferEvent")
	}
	if l.lastEvent.EventType != CONNECTING_TRANSFER_EVENT {
		t.Fatalf("Incorrect event type. Got %v. Expected %v", l.lastEvent.EventType, CONNECTING_TRANSFER_EVENT)
	}

	stats := l.lastEvent.SessionStats

	if stats.State != CONNECTING_SESSION_STATE {
		t.Fatalf("Incorrect state. Got %v. Expected %v.", stats.State, CONNECTING_SESSION_STATE)
	}
	if stats.Direction != "Send" {
		t.Fatalf("Incorrect direction. Got %v. Expected %v.", stats.Direction, "Send")
	}
	if stats.SessionId != "123456789" {
		t.Fatalf("Incorrect session id. Got %v. Expected %v.", stats.SessionId, "123456789")
	}

	fileStats := l.lastEvent.FileStats
	if fileStats != nil {
		t.Fatal("Expected no file statistics from INIT message.")
	}
}

func TestReceiveManagementMessage_SessionStart(t *testing.T) {
	sess, err := newTransferSession(testSessionOpts, testAscpOpts_NonPersistent)
	if err != nil {
		t.Fatal(err)
	}

	l := &testTransferEventListener{}
	sess.addListener(l)

	msg := "FASPMGR 2\n"
	msg += "Type: SESSION\n"
	msg += "Rate: 9999\n"
	msg += "MinRate: 1\n"
	msg += "Encryption: Yes\n"
	msg += "Adaptive: Fixed\n"
	msg += "Direction: Send\n"
	msg += "Port: 12345\n"
	msg += "Remote: Yes\n"
	msg += "Source: source.txt\n"
	msg += "Destination: destination.txt\n"
	msg += "Priority: 2\n"
	msg += "ServerNodeId: servNodeId\n"
	msg += "ClientNodeId: clientNodeId\n"
	msg += "ServerClusterId: servClusterId\n"
	msg += "ClientClusterId: clientClusterId\n"
	msg += "\n"

	err = sess.receiveManagementBlob(msg)
	if err != nil {
		t.Fatal(err)
	}

	if l.lastEvent == (TransferEvent{}) || l.lastEvent.SessionStats == nil {
		t.Fatal("listener did not receive TransferEvent")
	}
	if l.lastEvent.EventType != SESSION_START_TRANSFER_EVENT {
		t.Fatalf("Incorrect event type. Got %v. Expected %v.", l.lastEvent.EventType, SESSION_START_TRANSFER_EVENT)
	}

	stats := l.lastEvent.SessionStats

	if stats.State != STARTED_SESSION_STATE {
		t.Fatal("Incorrect state. Got %v. Expected %v.", stats.State, STARTED_SESSION_STATE)
	}
	if stats.TargetRateKbps != 9999 {
		t.Fatalf("Incorrect target rate. Got %v. Expected %v.", stats.TargetRateKbps, 9999)
	}
	if stats.MinRateKbps != 1 {
		t.Fatalf("Incorrect min rate. Got %v. Expected %v.", stats.MinRateKbps, 1)
	}
	if stats.Encryption != true {
		t.Fatalf("Incorrect encryption. Got %v. Expected %v", stats.Encryption, true)
	}
	if stats.Policy != FIXED_TRANSFER_POLICY {
		t.Fatalf("Incorrect poicy. Got %v. Expected %v.", stats.Policy, FIXED_TRANSFER_POLICY)
	}
	if stats.Direction != "Send" {
		t.Fatalf("Incorrect direction. Got %v. Expected %v.", stats.Direction, "Send")
	}
	if stats.UdpPort != 12345 {
		t.Fatalf("Incorrect Udp Port. Got %v. Expected %v.", stats.UdpPort, 12345)
	}
	if stats.Remote != true {
		t.Fatalf("Incorrect remote flag. Got %v. Expected %v.", stats.Remote, true)
	}
	if stats.SourcePaths != "source.txt" {
		t.Fatalf("Incorrect source paths. Got %v. Expected %v.", stats.SourcePaths, "source.txt")
	}
	if stats.DestPath != "destination.txt" {
		t.Fatalf("Incorrect destination paths. Got %v. Expected %v.", stats.DestPath, "destination.txt")
	}
	if stats.ServerNodeId != "servNodeId" {
		t.Fatalf("Incorrect server node id. Got %v. Expected %v.", stats.ServerNodeId, "servNodeId")
	}
	if stats.ClientNodeId != "clientNodeId" {
		t.Fatalf("Incorrect client node id. Got %v. Expected %v.", stats.ClientNodeId, "clientNodeId")
	}
	if stats.ServerClusterId != "servClusterId" {
		t.Fatalf("Incorrect server cluster id. Got %v. Expected %v.", stats.ServerClusterId, "servClusterId")
	}
	if stats.ClientClusterId != "clientClusterId" {
		t.Fatalf("Incorrect client cluster id. Got %v. Expected %v.", stats.ClientClusterId, "clientClusterId")
	}

	fileStats := l.lastEvent.FileStats
	if fileStats != nil {
		t.Fatal("Expected no file statistics from SESSION message.")
	}
}

func TestReceiveManagementMessage_Notification(t *testing.T) {
	sess, err := newTransferSession(testSessionOpts, testAscpOpts_NonPersistent)
	if err != nil {
		t.Fatal(err)
	}

	l := &testTransferEventListener{}
	sess.addListener(l)

	msg := "FASPMGR 2\n"
	msg += "Type: NOTIFICATION\n"
	msg += "Rate: 15000\n"
	msg += "MinRate: 10000\n"
	msg += "PMTU: 1492\n"
	msg += "Priorty: 2\n"
	msg += "\n"

	err = sess.receiveManagementBlob(msg)
	if err != nil {
		t.Fatal(err)
	}

	if l.lastEvent == (TransferEvent{}) || l.lastEvent.SessionStats == nil {
		t.Fatal("listener did not receive TransferEvent")
	}
	if l.lastEvent.EventType != RATE_MODIFICATION_TRANSFER_EVENT {
		t.Fatalf("Incorrect event type. Got %v. Expected %v.", l.lastEvent.EventType, RATE_MODIFICATION_TRANSFER_EVENT)
	}

	stats := l.lastEvent.SessionStats

	if stats.State != TRANSFERRING_SESSION_STATE {
		t.Fatalf("Incorrect state. Got %v. Expected %v.", stats.State, TRANSFERRING_FILE_STATE)
	}
	if stats.TargetRateKbps != 15000 {
		t.Fatalf("Incorrect target rate. Got %v. Expected %v.", stats.TargetRateKbps, 15000)
	}
	if stats.MinRateKbps != 10000 {
		t.Fatalf("Incorrect min rate. Got %v. Expected %v.", stats.MinRateKbps, 10000)
	}
	if stats.Pmtu != 1492 {
		t.Fatalf("Incorrect pmtu. Got %v. Expected %v.", stats.Pmtu, 1492)
	}

	fileStats := l.lastEvent.FileStats
	if fileStats != nil {
		t.Fatal("Expected no file statistics from NOTIFICATION message.")
	}
}

func TestReceiveManagementMessage_Stats(t *testing.T) {
	sess, err := newTransferSession(testSessionOpts, testAscpOpts_NonPersistent)
	if err != nil {
		t.Fatal(err)
	}

	l := &testTransferEventListener{}
	sess.addListener(l)

	msg := "FASPMGR 2\n"
	msg += "Type: STATS\n"
	msg += "File: some-file.txt\n"
	msg += "Size: 999\n"
	msg += "Loss: 100\n"
	msg += "TransferBytes: 500\n"
	msg += "FileBytes: 600\n"
	msg += "Delay: 10\n"
	msg += "\n"

	err = sess.receiveManagementBlob(msg)
	if err != nil {
		t.Fatal(err)
	}

	if l.lastEvent == (TransferEvent{}) || l.lastEvent.SessionStats == nil {
		t.Fatal("Listener did not receive transfer event.")
	}
	if l.lastEvent.EventType != FILE_START_TRANSFER_EVENT {
		t.Fatalf("Incorrect event type. Got %v. Expected %v.", l.lastEvent.EventType, FILE_START_TRANSFER_EVENT)
	}

	stats := l.lastEvent.SessionStats

	if stats.State != TRANSFERRING_SESSION_STATE {
		t.Fatalf("Incorrect state. Got %v. Expected %v.", stats.State, TRANSFERRING_FILE_STATE)
	}
	if stats.TotalTransferredBytes != 500 {
		t.Fatalf("Incorrect total transferred bytes. Got %v. Expected %v.", stats.TotalTransferredBytes, 500)
	}
	if stats.TotalWrittenBytes != 600 {
		t.Fatalf("Incorrect total written bytes. Got %v. Expected %v.", stats.TotalWrittenBytes, 600)
	}
	if stats.DelayMs != 10 {
		t.Fatalf("Incorrect delay. Got %v. Expected %v.", stats.DelayMs, 10)
	}

	fileStats := l.lastEvent.FileStats
	if fileStats == nil {
		t.Fatal("Expected file stats from STATS message.")
	}
	if fileStats.State != TRANSFERRING_FILE_STATE {
		t.Fatalf("Incorrect file state. Got %v. Expected %v.", fileStats.State, TRANSFERRING_FILE_STATE)
	}
	if fileStats.Name != "some-file.txt" {
		t.Fatalf("Incorrect file name. Got %v. Expected %v.", fileStats.Name, "some-file.txt")
	}
	if fileStats.SizeBytes != 999 {
		t.Fatalf("Incorrect file size. Got %v. Expected %v.", fileStats.SizeBytes, 999)
	}

}

func TestReceiveManagamentMessage_FileStop(t *testing.T) {
	sess, err := newTransferSession(testSessionOpts, testAscpOpts_NonPersistent)
	if err != nil {
		t.Fatal(err)
	}

	l := &testTransferEventListener{}
	sess.addListener(l)

	msg := "FASPMGR 2\n"
	msg += "Type: STOP\n"
	msg += "File: some-file.txt\n"
	msg += "Size: 1000\n"
	msg += "Written: 1000\n"
	msg += "Loss: 1\n"
	msg += "FileChecksum: 123456789\n"
	msg += "FileChecksumType: md5\n"
	msg += "\n"

	err = sess.receiveManagementBlob(msg)
	if err != nil {
		t.Fatal(err)
	}

	if l.lastEvent == (TransferEvent{}) || l.lastEvent.SessionStats == nil {
		t.Fatal("listener did not receive TransferEvent")
	}
	if l.lastEvent.EventType != FILE_STOP_TRANSFER_EVENT {
		t.Fatalf("Incorrect event type. Got %v. Expected %v.", l.lastEvent.EventType, FILE_STOP_TRANSFER_EVENT)
	}

	stats := l.lastEvent.SessionStats

	if stats.State != TRANSFERRING_SESSION_STATE {
		t.Fatalf("Incorrect state. Got %v. Expected %v.", stats.State, TRANSFERRING_FILE_STATE)
	}
	if stats.TotalLostBytes != 1 {
		t.Fatalf("Incorrect total bytes lost. Got %v. Expected %v.", stats.TotalLostBytes, 1)
	}
	if stats.FilesComplete != 1 {
		t.Fatalf("Incorrect files complete. Got %v. Expected %v.", stats.FilesComplete, 1)
	}

	fileStats := l.lastEvent.FileStats
	if fileStats == nil {
		t.Fatal("File stats should not be nil for STOP message.")
	}

	if fileStats.State != FINISHED_FILE_STATE {
		t.Fatalf("Incorrect file state. Got %v. Expected %v.", fileStats.State, FINISHED_FILE_STATE)
	}
	if fileStats.Name != "some-file.txt" {
		t.Fatalf("Incorrect file name. Got %v. Expected %v.", fileStats.Name, "some-file.txt")
	}
	if fileStats.SizeBytes != 1000 {
		t.Fatalf("Incorrect file size. Got %v. Expected %v.", fileStats.SizeBytes, 1000)
	}
	if fileStats.WrittenBytes != 1000 {
		t.Fatalf("Incorrect file written bytes. Got %v. Expected %v.", fileStats.WrittenBytes, 1000)
	}
	if fileStats.FileChecksum != "123456789" {
		t.Fatalf("Incorrect file checksum. Got %v. Expected %v.", fileStats.FileChecksum, "123456789")
	}
	if fileStats.FileChecksumType != "md5" {
		t.Fatalf("Incorrect file checksum type. Got %v. Expected %v.", fileStats.FileChecksumType, "md5")
	}

}

func TestReceiveManagementMessage_FileSkip(t *testing.T) {
	sess, err := newTransferSession(testSessionOpts, testAscpOpts_NonPersistent)
	if err != nil {
		t.Fatal(err)
	}

	l := &testTransferEventListener{}
	sess.addListener(l)

	msg := "FASPMGR2\n"
	msg += "Type: SKIP\n"
	msg += "File: afile.txt\n"
	msg += "\n"

	err = sess.receiveManagementBlob(msg)
	if err != nil {
		t.Fatal(err)
	}

	if l.lastEvent == (TransferEvent{}) || l.lastEvent.SessionStats == nil {
		t.Fatalf("Listener did not receive transfer event.")
	}
	if l.lastEvent.EventType != FILE_SKIP_TRANSFER_EVENT {
		t.Fatalf("Incorrect event type. Got %v. Expected %v.", l.lastEvent.EventType, FILE_SKIP_TRANSFER_EVENT)
	}

	stats := l.lastEvent.SessionStats

	if stats.State != TRANSFERRING_SESSION_STATE {
		t.Fatalf("Incorect state. Got %v. Expected %v.", stats.State, TRANSFERRING_SESSION_STATE)
	}
	if stats.FilesComplete != 1 {
		t.Fatalf("Incorrect files complete. Got %v. Expected %v.", stats.FilesComplete, 1)
	}
	if stats.FilesSkipped != 1 {
		t.Fatalf("Incorrect files skipped. Got %v. Expected %v.", stats.FilesSkipped, 1)
	}

	fileStats := l.lastEvent.FileStats
	if fileStats == nil {
		t.Fatal("File stats should not be nil for SKIP message.")
	}
	if fileStats.State != SKIPPED_FILE_STATE {
		t.Fatalf("Incorrect file state. Got %v. Expected %v.", fileStats.State, SKIPPED_FILE_STATE)
	}
	if fileStats.Name != "afile.txt" {
		t.Fatalf("Incorrect file name. Got %v. Expected %v.", fileStats.Name, "afile.txt")
	}
}

func TestReceiveManagementMessage_FileError(t *testing.T) {
	sess, err := newTransferSession(testSessionOpts, testAscpOpts_Persistent)
	if err != nil {
		t.Fatal(err)
	}

	l := &testTransferEventListener{}
	sess.addListener(l)

	msg := "FASPMGR 2\n"
	msg += "Type: FILEERROR\n"
	msg += "Code: 41\n"
	msg += "Description: bad file\n"
	msg += "File: badFile.txt\n"
	msg += "\n"

	err = sess.receiveManagementBlob(msg)
	if err != nil {
		t.Fatal(err)
	}

	if l.lastEvent == (TransferEvent{}) || l.lastEvent.SessionStats == nil {
		t.Fatalf("Listener did not receive transfer event.")
	}
	if l.lastEvent.EventType != FILE_ERROR_TRANSFER_EVENT {
		t.Fatalf("Incorrect event type. Got %v. Expected %v.", l.lastEvent.EventType, FILE_ERROR_TRANSFER_EVENT)
	}

	stats := l.lastEvent.SessionStats

	if stats.State != TRANSFERRING_SESSION_STATE {
		t.Fatalf("Incorect state. Got %v. Expected %v.", stats.State, TRANSFERRING_SESSION_STATE)
	}
	if stats.FilesFailed != 1 {
		t.Fatalf("Incorrect files failed. Got %v. Expected %v.", stats.FilesFailed, 1)
	}
	if stats.ErrorCode != 41 {
		t.Fatalf("Incorrect error code. Got %v. Expected %v.", stats.ErrorCode, 41)
	}
	if stats.ErrorDescription != "bad file" {
		t.Fatalf("Incorrect error descsrption. Got %v. Expected %v.", stats.ErrorDescription, "bad file")
	}

	fileStats := l.lastEvent.FileStats
	if fileStats == nil {
		t.Fatal("File stats should not be nil for FILEERROR message.")
	}
	if fileStats.State != FAILED_FILE_STATE {
		t.Fatalf("Incorrect file state. Got %v. Expected %v.", fileStats.State, FAILED_FILE_STATE)
	}
	if fileStats.Name != "badFile.txt" {
		t.Fatalf("Incorrect file name. Got %v. Expected %v.", fileStats.Name, "badFile.txt")
	}
	if fileStats.ErrorCode != 41 {
		t.Fatalf("Incorrect error code. Got %v. Expected %v.", stats.ErrorCode, 41)
	}
	if fileStats.ErrorDescription != "bad file" {
		t.Fatalf("Incorrect error descsrption. Got %v. Expected %v.", stats.ErrorDescription, "bad file")
	}
}

func TestReceiveManagementMessage_SessionDone(t *testing.T) {
	sess, err := newTransferSession(testSessionOpts, testAscpOpts_NonPersistent)
	if err != nil {
		t.Fatal(err)
	}

	l := &testTransferEventListener{}
	sess.addListener(l)

	msg := "FASPMGR 2\n"
	msg += "Type: DONE\n"
	msg += "ArgScansAttempted: 1\n"
	msg += "ArgScansCompleted: 1\n"
	msg += "PathScansAttempted: 1\n"
	msg += "PathScansFailed: 1\n"
	msg += "PathScansIrregular: 1\n"
	msg += "PathScansExcluded: 1\n"
	msg += "TransfersAttempted: 1\n"
	msg += "TransfersFailed: 1\n"
	msg += "TransfersPassed: 1\n"
	msg += "TransfersSkipped: 1\n"
	msg += "\n"

	err = sess.receiveManagementBlob(msg)
	if err != nil {
		t.Fatal(err)
	}

	if l.lastEvent == (TransferEvent{}) || l.lastEvent.SessionStats == nil {
		t.Fatalf("Listener did not receive transfer event.")
	}
	if l.lastEvent.EventType != SESSION_STOP_TRANSFER_EVENT {
		t.Fatalf("Incorrect event type. Got %v. Expected %v.", l.lastEvent.EventType, SESSION_STOP_TRANSFER_EVENT)
	}

	stats := l.lastEvent.SessionStats

	if stats.State != FINISHED_SESSION_STATE {
		t.Fatalf("Incorrect session state. Got %v. Expected %v.", stats.State, FINISHED_SESSION_STATE)
	}
	if stats.SourcePathsScanAttempted != 1 {
		t.Fatalf("Incorrect path scans attempted. Got %v. Expected %v.", stats.SourcePathsScanAttempted, 1)
	}
	if stats.SourcePathsScanFailed != 1 {
		t.Fatalf("Incorrect source path scan. Got %v. Expected %v.", stats.SourcePathsScanFailed, 1)
	}
	if stats.SourcePathsScanIrregular != 1 {
		t.Fatalf("Incorrect source path scan irregular. Got %v. Expected %v.", stats.SourcePathsScanIrregular, 1)
	}
	if stats.SourcePathsScanExcluded != 1 {
		t.Fatalf("Incorrect path scans excluded. Got %v. Expected %v.", stats.SourcePathsScanExcluded, 1)
	}
	if stats.TransfersAttempted != 1 {
		t.Fatalf("Incorrect transfers attempted. Got %v. Expected %v.", stats.TransfersAttempted, 1)
	}
	if stats.TransfersFailed != 1 {
		t.Fatalf("Incorrect transfers failed. Got %v. Expected %v.", stats.TransfersFailed, 1)
	}
	if stats.TransfersPassed != 1 {
		t.Fatalf("Incorrect transfers passed. Got %v. Expected %v.", stats.TransfersPassed, 1)
	}
	if stats.TransfersSkipped != 1 {
		t.Fatalf("Incorrect transfers skipped. Got %v. Expected %v.", stats.TransfersSkipped, 1)
	}

	fileStats := l.lastEvent.FileStats
	if fileStats != nil {
		t.Fatal("No file stats expected for ERROR message.")
	}

}

func TestReceiveManagementMessage_SessionError(t *testing.T) {
	sess, err := newTransferSession(testSessionOpts, testAscpOpts_NonPersistent)
	if err != nil {
		t.Fatal(err)
	}

	l := &testTransferEventListener{}
	sess.addListener(l)

	// Most of these fields are the same as in the "DONE" message.
	msg := "FASPMGR 2\n"
	msg += "Type: ERROR\n"
	msg += "Code: 1\n"
	msg += "Description: sess error\n"
	msg += "ArgScansAttempted: 1\n"
	msg += "ArgScansCompleted: 1\n"
	msg += "PathScansAttempted: 1\n"
	msg += "PathScansFailed: 1\n"
	msg += "PathScansIrregular: 1\n"
	msg += "PathScansExcluded: 1\n"
	msg += "TransfersAttempted: 1\n"
	msg += "TransfersFailed: 1\n"
	msg += "TransfersPassed: 1\n"
	msg += "TransfersSkipped: 1\n"
	msg += "\n"

	err = sess.receiveManagementBlob(msg)
	if err != nil {
		t.Fatal(err)
	}

	if l.lastEvent == (TransferEvent{}) || l.lastEvent.SessionStats == nil {
		t.Fatalf("Listener did not receive transfer event.")
	}
	if l.lastEvent.EventType != SESSION_ERROR_TRANSFER_EVENT {
		t.Fatalf("Incorrect event type. Got %v. Expected %v.", l.lastEvent.EventType, SESSION_ERROR_TRANSFER_EVENT)
	}

	stats := l.lastEvent.SessionStats

	if stats.State != FAILED_SESSION_STATE {
		t.Fatalf("Incorrect state. Got %v. Expected %v.", stats.State, FAILED_SESSION_STATE)
	}
	if stats.ErrorCode != 1 {
		t.Fatalf("Incorrect error code. Got %v. Expected %v.", stats.ErrorCode, 1)
	}
	if stats.ErrorDescription != "sess error" {
		t.Fatalf("Incorrect error description. Got %v. Expected %v.", stats.ErrorDescription, "sess error")
	}

	fileStats := l.lastEvent.FileStats
	if fileStats != nil {
		t.Fatal("No file stats expected for ERROR message.")
	}

}

func TestReceiveFragmentedManagementMessage(t *testing.T) {
	sess, err := newTransferSession(testSessionOpts, testAscpOpts_NonPersistent)
	if err != nil {
		t.Fatal(err)
	}

	l := &testTransferEventListener{}
	sess.addListener(l)

	firstFragment := "FASPMGR 2\n"
	firstFragment += "Type: NOTIFICATION\n"
	firstFragment += "Rate: 150"

	err = sess.receiveManagementBlob(firstFragment)
	if err != nil {
		t.Fatal(err)
	}

	if l.lastEvent != (TransferEvent{}) {
		t.Fatal("TransferSession should not dispatch a transfer event when " +
			"only a fragment of a management message has been received.")
	}

	secondFragment := "00\n"
	secondFragment += "MinRate: 10000\n"
	secondFragment += "PMTU: 1492\n"
	secondFragment += "Priorty: 2\n"
	secondFragment += "\n"

	err = sess.receiveManagementBlob(secondFragment)
	if err != nil {
		t.Fatal(err)
	}

	if l.lastEvent == (TransferEvent{}) || l.lastEvent.SessionStats == nil {
		t.Fatal("listener did not receive TransferEvent")
	}

	stats := l.lastEvent.SessionStats

	if stats.State != TRANSFERRING_SESSION_STATE {
		t.Fatalf("Incorrect state. Got %v. Expected %v.", stats.State, TRANSFERRING_FILE_STATE)
	}
	if stats.TargetRateKbps != 15000 {
		t.Fatalf("Incorrect target rate. Got %v. Expected %v.", stats.TargetRateKbps, 15000)
	}
	if stats.MinRateKbps != 10000 {
		t.Fatalf("Incorrect min rate. Got %v. Expected %v.", stats.MinRateKbps, 10000)
	}
	if stats.Pmtu != 1492 {
		t.Fatalf("Incorrect pmtu. Got %v. Expected %v.", stats.Pmtu, 1492)
	}

	fileStats := l.lastEvent.FileStats
	if fileStats != nil {
		t.Fatal("Expected no file statistics from NOTIFICATION message.")
	}

}

func TestReceiveInvalidManagementMessage(t *testing.T) {
	sess, err := newTransferSession(testSessionOpts, testAscpOpts_NonPersistent)
	if err != nil {
		t.Fatal(err)
	}

	l := &testTransferEventListener{}
	sess.addListener(l)

	msg := "FASPMGR 2\n"
	msg += "invalid\n"
	msg += "\n"

	err = sess.receiveManagementBlob(msg)
	if err == nil {
		t.Fatal("No error returned when invalid managment message given.")
	}
}
