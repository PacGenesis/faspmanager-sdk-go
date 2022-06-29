package faspmanager

import (
	"fmt"
	"time"
)

// A TransferEvent contains information about about an ongoing transfer.
type TransferEvent struct {

	// EventType is the type of event.
	EventType TransferEventType

	// SessionStats contains statistics concerning the transfer session
	// itself. Will never be nil.
	SessionStats *SessionStatistics

	// FileStats contains information about one file in the transfer. Note that
	// this may be nil if a TransferEvent carries no information about a particular
	// file.
	FileStats *FileStatistics
}

func (t TransferEvent) String() string {
	return fmt.Sprintf("{EventType: %s, SessionStats: %+v, FileStats: %+v}",
		t.EventType, t.SessionStats, t.FileStats)
}

// TransferListener must be implemented to subscribe to TransferEvents.
type TransferListener interface {

	// OnTransferEvent is called on a worker goroutine whenever a TransferEvent
	// occurs.
	OnTransferEvent(TransferEvent)
}

type TransferEventType string

const (
	// The transfer session is connecting to the remote host.
	CONNECTING_TRANSFER_EVENT TransferEventType = "Connecting"

	// The transfer session has begun.
	SESSION_START_TRANSFER_EVENT TransferEventType = "Session Start"

	// The rate of the transfer has changed. This always occurs once at the start of a transfer
	// and may occur again if the transfer rate is altered.
	RATE_MODIFICATION_TRANSFER_EVENT TransferEventType = "Rate Modification"

	// A specific file transfer has begun.
	FILE_START_TRANSFER_EVENT TransferEventType = "File Start"

	// An update on the state of an ongoing transfer.
	PROGRESS_TRANSFER_EVENT TransferEventType = "Progress"

	// A specific file transfer has completed or otherwise stopped.
	FILE_STOP_TRANSFER_EVENT TransferEventType = "File Stop"

	// A specific file transfer has encountered an error.
	FILE_ERROR_TRANSFER_EVENT TransferEventType = "File Error"

	// The transfer session has encountered an error.
	SESSION_ERROR_TRANSFER_EVENT TransferEventType = "Session Error"

	// The transfer session has stopped.
	SESSION_STOP_TRANSFER_EVENT TransferEventType = "Session Stop"

	// A file has been skipped. Note that TransferOptions.ReportSkippedFiles must be set
	// for this event to be triggered.
	FILE_SKIP_TRANSFER_EVENT TransferEventType = "File Skip"
)

// SessionStatistics contains information about a transfer session.
type SessionStatistics struct {

	// The session ID.
	Id SessionId

	// The identifier used by ASCP for this session.
	SessionId string

	// The transfer retry, as specified in TransferOptions.
	TransferRetry int

	// State of this session.
	State SessionStateType

	// The source paths used in this session.
	SourcePaths string

	// The destination paths used in this session.
	DestPath string

	// The UDP Port used for this session.
	UdpPort int

	// The target rate of this session.
	TargetRateKbps int

	// The min-rate of this session.
	MinRateKbps int

	// Whether encryption has been enabled or disabled for this session.
	Encryption bool

	// The current TransferPolicyType of this session.
	Policy TransferPolicyType

	// Total number of bytes written at the destination
	// If this is a resumed transfer, this field
	// includes bytes transferred during previous sessions.
	TotalWrittenBytes uint64

	// Number of files transferred during this session.
	FilesComplete int

	// Number of files skipped during this session.
	FilesSkipped int

	// Number of files which failed to transfer during this session.
	FilesFailed int

	// Error code if this session failed.
	ErrorCode int

	// Error description if this session failed.
	ErrorDescription string

	// The bandwidth cap enforced on this session.
	BandwidthCapKbps int

	// The session cookie from this session's TransferOptions.
	Cookie string

	// The remote host.
	Host string

	// The time at which this session started.
	StartTime time.Time

	// Number of bytes lost over the course of this session.
	TotalLostBytes uint64

	// Number of bytes transferred over the course of this session.
	TotalTransferredBytes uint64

	// The session's token from this session's TransferOptions.
	Token string

	// The username used to connect to the remote host.
	User string

	// Measured bandwidth, if automatic bandwidth detection is turned on.
	MeasuredLinkRateKbps int

	// Size of all files in this session, calculated before the start of the transfer.
	PreCalcTotalBytes int

	// Number of files in this session, calculated before the start of the transfer.
	PreCalcTotalFiles int

	// The direction of the transfer (send or receive).
	Direction string

	// True if this session was initiated by a remote host.
	Remote bool

	// True if this session was not initiated by this instance of the Manager.
	//OtherInitiated           bool

	// Network delay.
	DelayMs int

	// Elapsed microseconds since the beginning of this transfer.
	ElapsedUSec int

	// Path to the manifest file, if generated. Available only at the end of the session.
	ManifestFilePath string

	// The count of file transfers with errors.
	TransfersAttempted int

	// The count of failed transfers with errors.
	TransfersFailed int

	// The count of file transfers successfully completed.
	TransfersPassed int

	// The count of file transfers skipped (e.g. if the file was already at the destination,
	// overwrite policy violation).
	TransfersSkipped int

	// Source paths attempted.
	SourcePathsScanAttempted int

	// Source paths failed.
	SourcePathsScanFailed int

	// Source paths pointing to irregular files.
	SourcePathsScanIrregular int

	// Source paths excluded due to matching exclude arguments.
	SourcePathsScanExcluded int

	// The Path MTU.
	Pmtu int

	// The Fasp TCP port for authentication.
	TcpPort int

	// The server node ID.
	ServerNodeId string

	// The client node ID.
	ClientNodeId string

	// The server cluster ID.
	ServerClusterId string

	// The client cluster ID.
	ClientClusterId string

	// The informative retry timeout.
	RetryTimeout int
}

// FileStatistics contains information about a single file transferred in a transfer session.
type FileStatistics struct {

	// The number of contiguous bytes of a file on the disk. Any other
	// application that needs to consume the file being transferred need
	// not wait for the transfer to complete. Instead it can consume the
	// number of bytes indicated by this field, from the beginning of the file.
	ContiguousBytes int64

	// Error code if State is FAILED_FILE_STATE.
	ErrorCode int

	// Error description if State is FAILED_FILE_STATE.
	ErrorDescription string

	// The name of the file.
	Name string

	// The file size.
	SizeBytes int64

	// The state of the file.
	State FileStateType

	// The number of the file's bytes written to the destination. Not necessarily contiguous.
	WrittenBytes int64

	// Starting offset in the file (either when a broken transfer is resumed or if
	// only a range of bytes is requested) Non zero only if the transfer is a
	// resume or if a range of bytes is requested.
	StartByte int64

	// Offset in the file where the transfer is to end.  Non-zero only if a range
	// of bytes is requested.
	EndByte int64

	// The file checksum type.
	FileChecksumType string

	// The file checksum.
	FileChecksum string
}

// SessionStateType describes the state of a transfer session.
type SessionStateType string

const (
	// The session has been submitted and is about to begin.
	SUBMITTED_SESSION_STATE = SessionStateType("Submitted")

	// The session is attempting to authenticate with the remote endpoint.
	AUTHENTICATING_SESSION_STATE = SessionStateType("Authenticating")

	// The session is connecting with the remote endpoint.
	CONNECTING_SESSION_STATE = SessionStateType("Connecting")

	// A connection was established and the session has started.
	STARTED_SESSION_STATE = SessionStateType("Started")

	// The session is in progress.
	TRANSFERRING_SESSION_STATE = SessionStateType("Transferring")

	// The session completed successfully.
	FINISHED_SESSION_STATE = SessionStateType("Finished")

	// An error occurred that prevented the session from completing.
	FAILED_SESSION_STATE = SessionStateType("Failed")
)

// FileStateType describes the state of a transfer of a single file.
type FileStateType string

const (
	// An error occurred while transferring the file.
	FAILED_FILE_STATE = FileStateType("Failed")

	// The file transferred successfully.
	FINISHED_FILE_STATE = FileStateType("Finished")

	// The file is currently transferring.
	TRANSFERRING_FILE_STATE = FileStateType("Transferring")

	// The file was skipped during the transfer.
	SKIPPED_FILE_STATE = FileStateType("Skipped")
)
