package faspmanager

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strconv"
)

// TransferOptions carry information about a requested transfer. An empty or nil
// TransferOptions struct specifies sensible defaults, so options need only be
// explicitly specified when non-defaults are desired.
type TransferOptions struct {

	// Automatically detect bandwidth capacity. Default: false This feature is
	// known to be unreliable for connections faster than 10Mbps.  You are
	// discouraged from using it.
	AutoDetectCapacity bool

	// An arbitrary field used for application-specific requirements.
	Cookie string

	// Create target directory if it doesn't already exist. Default: false
	CreateDirs bool

	// Sets the write size for a stream-to-file transfer.
	ChunkSize int

	// Specify the datagram size (MTU) for fasp to use.
	// Otherwise fasp will automatically use the detected path MTU.
	DatagramSize int

	// The encryption cipher to use.
	Cipher CipherType

	// Set this to true to disable encryption.
	DisableEncryption bool

	// Specify a directory to write the log file in for the local host, instead
	// of using the default directory.
	LocalLogDir string

	// The minimum transfer rate in kbps. This value is interpreted as a
	// percentage of detected bandwidth when autoDetectCapacity is true.
	MinRateKbps int

	// Transfer Rate Policy. Default: FAIR_TRANSFER_POLICY
	Policy TransferPolicyType

	// Preserve file date attributes. Default: false.
	PreserveDates bool

	// Specify a directory to write the log file in for the local host, instead of
	// using the default directory.
	RemoteLogDir string

	// Informs any monitoring applications how long you intend to retry. Note that
	// the fasp manager itself does not implement retry; you must do this.
	// Default: 0 (no retry)
	RetryTimeoutS int

	// How symbolic links are handled.
	SymLinkPolicy SymLinkPolicyType

	// Resume policy for partially transferred files.
	ResumeCheck ResumeType

	// Specify the prefix to be stripped off from each source object.
	// The remaining portion of the source path is kept intact at the destination.
	// If the target directory does not exist, this option must be used
	// along with TransferOptions.CreateDirs.
	SourceBase string

	// The target transfer rate in kbps. This value is interpreted as a
	// percentage of detected bandwidth when autoDetectCapacity is true.
	// Default: 10000 (Or 100% when autoDetectCapacity is turned on).
	TargetRateKbps int

	// The TCP port used for transfer initialization. Default: 22
	TcpPort int

	// Security token.
	Token string

	// The UDP port used for transferring data. Default: 33001
	UdpPort int

	// Specifying a passphrase turns on content protection. With content protection
	// uploaded files are left encrypted on the destination and downloads are
	// decrypted as they as downloaded.
	ContentProtectionPassphrase string

	// Before starting the transfer, calculate and report the aggregate size of all
	// files in the session. For large file sets, this option can introduce a
	// significant delay. Use caution!  Default: false
	PreCalculateJobSize bool

	// Set this to a value other than 'NONE' to enable file manifests
	FileManifestFormat FileManifestFormatType

	// Path to a directory on the client machine where the manifest file is to be
	// written.  If the path is invalid the manifest file will not be generated.
	FileManifestPath string

	// Delete source files after successful transfer.
	DeleteSource DeleteSourceType

	// Remove source files after transferring. Default: false
	RemoveFileAfterTransfer bool

	// Remove source empty directories after transferring. Default: false
	RemoveEmptyDirectories bool

	// Skip special files. Default: false
	SkipSpecialFiles bool

	// Preserve file owner UID. Default: false
	PreserveFileOwnerUid bool

	// Preserve file owner GID. Default: false
	PreserveFileOwnerGid bool

	// Overwrite behavior when a file being transferred already exists
	// at the destination. Default: DIFFERENT_OVERWRITE_POLICY
	OverwritePolicy OverwritePolicyType

	// Read block size in bytes. Must be a positive number. 0 specifies the default.
	ReadSize int

	// Write block size in bytes. Must be a positive number. 0 specifies the default.
	WriteSize int

	//  Pre and Post command file path.
	PrePostCommandPath string

	// Alternate Configuration file name.
	AlternateConfigFileName string

	// Default suffix is ".partial"
	PartialFileSuffix string

	// Exclude files newer than "mtime".
	// Exclude files from the transfer based on when the file was last changed.
	// Positive MTIME values are compared directly to the "mtime" timestamp
	// in the source file system, usually seconds since 1970-01-01 00:00:00.
	// Negative MTIME values are taken as specifying a file "mtime" value
	// equal to the present + MTIME.  Some file servers use a different mtime
	// reference point, and then only positive MTIME argument values relative
	// to that reference will work sensibly.
	ExcludeNewerThan int

	// Exclude files older than "mtime".
	// Exclude files from the transfer based on when the file was last changed.
	// Positive MTIME values are compared directly to the "mtime" timestamp
	// in the source file system, usually seconds since 1970-01-01 00:00:00.
	// Negative MTIME values are taken as specifying a file "mtime" value
	// equal to the present + MTIME.  Some file servers use a different mtime
	// reference point, and then only positive MTIME argument values relative
	// to that reference will work sensibly.
	ExcludeOlderThan int

	// If a transfer would result in an existing file being overwritten, move that file
	// to filename.yyyy.mm.dd.hh.mm.ss.index in the same directory(where index is set to 1
	// at the beginning of each new second and incremented for each file saved in this manner
	// during the same second). File attributes are maintained in the renamed file.
	SaveBeforeOverwriteEnabled bool

	// Check it against server SSH host key fingerprint (e.g.,
	// f74e5de9ed0d62feaf0616ed1e851133c42a0082).
	CheckSshFingerprint string

	// Adjust the maximum size in bytes of a retransmission request (max 1440).
	RetransmissionRequestMaxSize int

	// List of patterns (maximum 16) used to exclude files from transferring.
	// Two special symbols are accepted as part of a pattern: * to match any
	// character zero or more times, and ? to match any character exactly once.
	ExcludePatterns []string

	// The Http fallback.
	HttpFallback *HttpFallback

	// Move transferred files to this location after transfer success.
	MoveAfterTransferPath string

	// Specify the destination root directory of the transfers.
	DestinationRoot string

	// Set to true to include skipped files in the stats received by TransferListeners.
	// By default, these are not reported in the TransferEvent.
	ReportSkippedFiles bool

	// Controls whether the checksum of each file is reported and what
	// type of checksum is reported.
	FileChecksum FileChecksumType

	// The transfer retry.
	TransferRetry int

	// A transfer identifier. By default, this is a randomly generated UUID.
	TransferId string

	// Preserve extended attributes on the local host when transferring files
	// between different types of file systems.
	PreserveXAttrs PreserveModeType

	// Preserve extended attributes on the remote host when transferring files
	// between different types of file systems. If not specified, defaults to the
	// same as TransferOptions.PreserveXAttrs.
	RemotePreserveXAttrs PreserveModeType

	// Preserve access control lists on the local lost when transferring files
	// between different types of file systems.
	PreserveAcls PreserveModeType

	// Preserve access control lists on the remote host when transferring files
	// between different types of file systems. If not specified, defaults to the
	// same as TransferOptions.PreserveAcls.
	RemotePreserveAcls PreserveModeType

	// Delete files that exist at the destination but not at the source, before any
	// files are transferred.  Do not use this option with multiple sources,
	// keepalive, or HTTP fallback. Default: false
	DeleteBeforeTransfer bool

	// Set the file/directory creation time at the destination to that of the
	// source.  Available on Windows clients only. If the destination is a
	// non-Windows host, this option is ignored.  (Note: Do not confuse this with
	// UNIX ctime, which represents "change time", indicating the time when
	// metadata was last updated.). Default: false.
	PreserveCreationTime bool

	// Set the file/directory modification time at the destination to that of the
	// source. Default: false
	PreserveModificationTime bool

	// Set the file/directory access time (the last time the file was read or
	// written) at the destination to that of the source. This results in the
	// destination file having the access time that the source file had prior to
	// the copy operation. The act of copying the source file to the destination
	// results in an update to the source file's access time. Default: false
	PreserveAccessTime bool

	// Restore the access time of the file at the source once the copy operation is
	// complete (because the file system at the source regards the transfer
	// operation as an access). Default: false
	PreserveSourceAccessTime bool

	// Do not check for duplicate files. Default: false
	SkipDirTraversalDupes bool

	// Apply the local docroot. Default: false
	ApplyLocalDocroot bool

	// This option augments multi-process transfers,(also known as parallel
	// transfers).  If the size of the files to be transferred is greater than or
	// equal to this value threshold, files will be split.
	MultiSessionThreshold int

	// If prompted to accept a host key when connecting to a remote host,ignores
	// the request.  Default: false
	IgnoreHostKey bool

	// Add prefix to the beginning of each source path. This can be either a
	// conventional path or a URI.  However, it can only be a URI if there is no
	// root defined.
	SourcePrefix string

	// Remove the source directory argument itself. For use with
	// TransferOptions.RemoveEmptyDirectories.  Default: false.
	RemoveEmptySourceDirectory bool

	// ExtraOptions is an optional slice of additional command line parameters to
	// pass to ASCP on the start of a transfer.
	ExtraOptions []string

	// ASCP4 doesn't support certain options. This is set internally to signify
	// those options cannot be used.
	ascp4StreamFlag bool
}

// HttpFallback contains options for HTTP transfers when FASP's UDP connection
// fails.
type HttpFallback struct {

	// Encode all HTTP transfers as JPEG files. Default: false
	EncodeAllAsJpeg bool

	// The HTTPS transfer's key file name.
	HttpsKeyFileName string

	// The HTTPS certificate's file name
	HttpsCertFileName string

	// The port for HTTP fallback transfers.
	HttpPort int

	// The proxy server address used by HTTP fallback.
	HttpProxyAddressHost string

	// The port of the HTTP proxy server used by HTTP fallback.
	HttpProxyAddressPort int
}

func (h *HttpFallback) getCommandLineStrings() []string {
	var args []string
	args = append(args, "-y1")
	if h.EncodeAllAsJpeg {
		args = append(args, "-j1")
	}
	if h.HttpPort > 0 {
		args = append(args, "-t"+strconv.Itoa(h.HttpPort))
	}
	if len(h.HttpsKeyFileName) > 0 {
		args = append(args, "-Y"+h.HttpsKeyFileName)
	}
	if len(h.HttpsCertFileName) > 0 {
		args = append(args, "-I"+h.HttpsCertFileName)
	}
	if len(h.HttpProxyAddressHost) > 0 {
		proxyArg := "-x" + h.HttpProxyAddressHost
		if h.HttpProxyAddressPort > 0 {
			proxyArg = proxyArg + ":" + strconv.Itoa(h.HttpProxyAddressPort)
		}
		args = append(args, proxyArg)
	}
	return args
}

func (h *HttpFallback) validate() error {
	if len(h.HttpsKeyFileName) > 0 || len(h.HttpsCertFileName) > 0 {
		if _, err := os.Stat(h.HttpsKeyFileName); os.IsNotExist(err) {
			return errors.New("TransferOptions: Given HttpFallback HttpsKeyFileName does not exist.")
		}
		if _, err := os.Stat(h.HttpsCertFileName); os.IsNotExist(err) {
			return errors.New("TransferOptions: Given HttpFallback HttpsCertFileName does not exist.")
		}
	}
	if h.HttpPort < 0 {
		return errors.New("TransferOptions: If given, HttpFallback HttpPort must be non-negative. Was: " +
			strconv.Itoa(h.HttpPort))
	}
	if h.HttpProxyAddressPort < 0 {
		return errors.New("TransferOptions: If given, HttpFallback HttpProxyAddress Port must be non-negative. Was: " +
			strconv.Itoa(h.HttpProxyAddressPort))
	}
	return nil
}

// Rate policy option for a FASP transfer.
type TransferPolicyType int

const (
	// Transfer at the target rate but slow down to allow other traffic.
	FAIR_TRANSFER_POLICY TransferPolicyType = iota

	// Transfer at the target rate regardless of network traffic.
	FIXED_TRANSFER_POLICY

	// Transfer at the target rate but slow down to allow other traffic.
	// Takes higher priority than FAIR_TRANSFER_POLICY.
	HIGH_TRANSFER_POLICY

	// Use only unused network bandwidth.
	LOW_TRANSFER_POLICY
)

func (t TransferPolicyType) getCommandLineString() (string, error) {
	switch t {
	case FAIR_TRANSFER_POLICY:
		return "fair", nil
	case FIXED_TRANSFER_POLICY:
		return "fixed", nil
	case HIGH_TRANSFER_POLICY:
		return "high", nil
	case LOW_TRANSFER_POLICY:
		return "low", nil
	default:
		return "", fmt.Errorf("Unrecognized TransferPolicyType: %v", t)
	}
}

func (t TransferPolicyType) getManagementMessageString() (string, error) {
	switch t {
	case FAIR_TRANSFER_POLICY:
		return "Adaptive", nil
	case FIXED_TRANSFER_POLICY:
		return "Fixed", nil
	case HIGH_TRANSFER_POLICY:
		return "Adaptive", nil
	case LOW_TRANSFER_POLICY:
		return "Trickle", nil
	default:
		return "", fmt.Errorf("Unrecognized TransferPolicyType: %v", t)
	}
}

// Encryption cipher type for transfers.
type CipherType int

const (
	// The default when no cipher is given.
	UNINITIALIZED_CIPHER CipherType = iota

	// Use an aes128 cipher.
	AES_128_CIPHER

	// Use an aes192 cipher.
	AES_192_CIPHER

	// Use an aes256 cipher.
	AES_256_CIPHER
)

func (t CipherType) getCommandLineString() (string, error) {
	switch t {
	case AES_128_CIPHER:
		return "aes128", nil
	case AES_192_CIPHER:
		return "aes192", nil
	case AES_256_CIPHER:
		return "aes256", nil
	default:
		return "", fmt.Errorf("Unrecognized CipherType: %v", t)
	}
}

// Policy to handle symbolic links in the filesystem.
type SymLinkPolicyType int

const (
	// The default when no policy is given.
	UNINITIALIZED_SYM_LINK_POLICY SymLinkPolicyType = iota

	// Copy the symbolic link itself to the destination.
	COPY_SYM_LINK_POLICY

	// Follow the symbolic link and copy the file.
	FOLLOW_SYM_LINK_POLICY

	// Forcibly copy the symbolic link itself.
	COPY_FORCE_SYM_LINK_POLICY

	// Skip the symbolic link.
	SKIP_SYM_LINK_POLICY
)

func (t SymLinkPolicyType) getCommandLineString() (string, error) {
	switch t {
	case COPY_SYM_LINK_POLICY:
		return "copy", nil
	case FOLLOW_SYM_LINK_POLICY:
		return "follow", nil
	case SKIP_SYM_LINK_POLICY:
		return "skip", nil
	case COPY_FORCE_SYM_LINK_POLICY:
		return "copy+force", nil
	default:
		return "", fmt.Errorf("Unrecognized SymLinkPolicyType: %v", t)
	}
}

// Resume options for transferred files if a file is found with the same name as at the destination.
type ResumeType int

const (
	// The default when no ResumeType is given.
	UNINITIALIZED_RESUME ResumeType = iota

	// Always re-transfer.
	OFF_RESUME

	// If the size of the source and destination match, resume the transfer.
	FILE_ATTRIBUTES_RESUME

	// If the sparse checksum of the source and destination match, resume the transfer.
	SPARSE_CHECKSUM_RESUME

	// If the full checksum of the source and destination match, resume the transfer.
	FULL_CHECKSUM_RESUME
)

func (t ResumeType) getCommandLineString() (string, error) {
	switch t {
	case OFF_RESUME:
		return "0", nil
	case FILE_ATTRIBUTES_RESUME:
		return "1", nil
	case SPARSE_CHECKSUM_RESUME:
		return "2", nil
	case FULL_CHECKSUM_RESUME:
		return "3", nil
	default:
		return "", fmt.Errorf("Unrecognized ResumeType: %v", t)
	}
}

// The format of the file manifest.
type FileManifestFormatType int

const (
	// Disable generation of the manifest.
	NONE_FILE_MANIFEST_FORMAT FileManifestFormatType = iota

	// Generate a plain text manifest file.
	TEXT_FILE_MANIFEST_FORMAT
)

// Policy to specify deletion of source files.
type DeleteSourceType int

const (
	// Disable source deletion.
	NONE_DELETE_SOURCE DeleteSourceType = iota

	// Delete source files after a successful transfer but not directories.
	FILES_ONLY_DELETE_SOURCE

	// Delete source files as well as directories after a successsful transfer.
	FILES_AND_DIRECTORIES_DELETE_SOURCE

	// Delete empty directories after a successful transfer.
	EMPTY_DIRECTORIES_DELETE_SOURCE
)

// Policy to handle overwrites of existing files.
type OverwritePolicyType int

const (
	// Overwrite only if the existing file is different.
	DIFFERENT_OVERWRITE_POLICY OverwritePolicyType = iota

	// Always overwrite the existing file.
	ALWAYS_OVERWRITE_POLICY

	// Overwrite only if the existing file is different and older.
	DIFFERENT_AND_OLDER_OVERWRITE_POLICY

	// Never overwrite an existing file.
	NEVER_OVERWRITE_POLICY

	// Overwrite only if the existing file is older.
	OLDER_OVERWRITE_POLICY
)

func (t OverwritePolicyType) getCommandLineString() (string, error) {
	switch t {
	case DIFFERENT_OVERWRITE_POLICY:
		return "diff", nil
	case ALWAYS_OVERWRITE_POLICY:
		return "always", nil
	case DIFFERENT_AND_OLDER_OVERWRITE_POLICY:
		return "diff+older", nil
	case NEVER_OVERWRITE_POLICY:
		return "never", nil
	case OLDER_OVERWRITE_POLICY:
		return "older", nil
	default:
		return "", fmt.Errorf("Unrecognized OverwritePolicyType: %v", t)
	}
}

// PreserveModeType describes the way in which extended attributes or access
// control lists are preserved when transferring files between hosts with different
// types of file systems.
type PreserveModeType int

const (

	// Extended attributes and access control lists are not preserved. This mode is
	// supported on all file systems and is the default.
	NONE_PRESERVE_MODE = iota

	// Extended attributes and access control lists are preserved using native
	// capabilities of the file system. This mode is not supported on all file
	// systems.
	NATIVE_PRESERVE_MODE

	// Extended attributes and access control lists for a file (say, readme.txt)
	// are preserved in a second file, whose name is composed of the name of the
	// primary file with .aspera-meta appended to it; for example,
	// readme.txt.aspera-meta. The Aspera metafiles are platform independent and
	// can be copied between hosts without loss of information. This storage mode
	// is supported on all file systems.
	METAFILE_PRESERVE_MODE
)

func (t PreserveModeType) getCommandLineString() (string, error) {
	switch t {
	case NONE_PRESERVE_MODE:
		return "none", nil
	case NATIVE_PRESERVE_MODE:
		return "native", nil
	case METAFILE_PRESERVE_MODE:
		return "metafile", nil
	default:
		return "", fmt.Errorf("Unrecognized PreserveModeType: %v\n", t)
	}
}

type FileChecksumType int

const (

	// Don't report the file checksum.
	NONE_CHECKSUM = iota

	// Report the checksum as sha512.
	SHA512_CHECKSUM

	// Report the checksum as sha384.
	SHA384_CHECKSUM

	// Report the checksum as sha256.
	SHA256_CHECKSUM

	// Report the checksum as sha1.
	SHA1_CHECKSUM

	// Report the checksum as md5.
	MD5_CHECKSUM
)

func (t FileChecksumType) getCommandLineString() (string, error) {
	switch t {
	case SHA512_CHECKSUM:
		return "sha-512", nil
	case SHA384_CHECKSUM:
		return "sha-384", nil
	case SHA256_CHECKSUM:
		return "sha-256", nil
	case SHA1_CHECKSUM:
		return "sha1", nil
	case MD5_CHECKSUM:
		return "md5", nil
	default:
		return "", fmt.Errorf("Unrecognized FileChecksumType: %v\n", t)
	}
}

// Defaults.
const (
	DefaultDestRoot   = "."
	DefaultTargetRate = 10000
	DefaultMinRate    = 0
	DefaultChunkSize  = 1024 * 1024 // 1MB
	DefaultUdpPort    = 33001
	DefaultTcpPort    = 22
)

func (o *TransferOptions) getTargetRateOrSetDefault() int {
	if o.TargetRateKbps == 0 {
		if o.AutoDetectCapacity {
			o.TargetRateKbps = 100
		} else {
			o.TargetRateKbps = DefaultTargetRate
		}
	}
	return o.TargetRateKbps
}

func (o *TransferOptions) getMinRateOrSetDefault() int {
	if o.MinRateKbps == 0 {
		if o.AutoDetectCapacity {
			o.MinRateKbps = 100
		}
		o.MinRateKbps = DefaultMinRate
	}
	return o.MinRateKbps
}

func (o *TransferOptions) getChunkSizeOrSetDefault() int {
	if o.ChunkSize == 0 {
		o.ChunkSize = DefaultChunkSize
	}
	return o.ChunkSize
}

func (o *TransferOptions) getUdpPortOrSetDefault() int {
	if o.UdpPort == 0 {
		o.UdpPort = DefaultUdpPort
	}
	return o.UdpPort
}

func (o *TransferOptions) getTcpPortOrSetDefault() int {
	if o.TcpPort == 0 {
		o.TcpPort = DefaultTcpPort
	}
	return o.TcpPort
}

func (o *TransferOptions) getTransferIdOrSetDefault() string {
	if len(o.TransferId) == 0 {
		o.TransferId = generateUniqueId()
	}
	return o.TransferId
}

// Used to encode TransferId and TransferRetry options as JSON.
type asperaJsonOptions struct {
	Inner innerAsperaJsonOptions `json:"aspera"`
}

type innerAsperaJsonOptions struct {
	XferId    string `json:"xfer_id"`
	XferRetry int    `json:"xfer_retry"`
}

// Returns a slice corresponding to the ASCP command line options built from these
// transfer options.  Does not validate the options.
func (o *TransferOptions) getCommandLineParams() (args []string) {
	if o.AutoDetectCapacity {
		args = append(args, "-wf")
	}
	if o.Cipher != UNINITIALIZED_CIPHER {
		if cstr, err := o.Cipher.getCommandLineString(); err == nil {
			args = append(args, "-c"+cstr)
		}
	} else if o.DisableEncryption {
		args = append(args, "-T")
	}
	if policy, err := o.Policy.getCommandLineString(); err == nil {
		args = append(args, "--policy="+policy)
	}

	args = append(args, "-O "+strconv.Itoa(o.getUdpPortOrSetDefault()))
	args = append(args, "-P "+strconv.Itoa(o.getTcpPortOrSetDefault()))
	args = append(args, "--retry-timeout="+strconv.Itoa(o.RetryTimeoutS))

	// TODO: Remove this when ASCP 4 adds support.
	if !o.ascp4StreamFlag {
		if o.FileManifestFormat == NONE_FILE_MANIFEST_FORMAT {
			args = append(args, "--file-manifest=none")
		} else if o.FileManifestFormat == TEXT_FILE_MANIFEST_FORMAT {
			args = append(args, "--file-manifest=text")
			if o.FileManifestPath != "" {
				args = append(args, "--file-manifest-path="+o.FileManifestPath)
			}
		}
	}

	autoDetectStr := ""
	if o.AutoDetectCapacity {
		autoDetectStr = "%"
	}
	args = append(args, "-l"+strconv.Itoa(o.getTargetRateOrSetDefault())+autoDetectStr)
	if o.MinRateKbps != 0 {
		args = append(args, "-m"+strconv.Itoa(int(o.getMinRateOrSetDefault()))+autoDetectStr)
	}

	if o.SymLinkPolicy != UNINITIALIZED_SYM_LINK_POLICY {
		if s, err := o.SymLinkPolicy.getCommandLineString(); err == nil {
			args = append(args, "--symbolic-links="+s)
		}
	}
	if o.ResumeCheck != UNINITIALIZED_RESUME {
		if s, err := o.ResumeCheck.getCommandLineString(); err == nil {
			args = append(args, "-k"+s)
		}
	}
	if o.PreserveDates {
		args = append(args, "-p")
	}
	if o.SkipSpecialFiles {
		args = append(args, "--skip-special-files")
	}
	if o.PreserveFileOwnerGid {
		args = append(args, "--preserve-file-owner-uid")
	}
	if o.PreserveFileOwnerGid {
		args = append(args, "--preserve-file-owner-gid")
	}
	if o.DatagramSize != 0 {
		args = append(args, "-Z"+strconv.Itoa(o.DatagramSize))
	}
	if o.CreateDirs {
		args = append(args, "-d")
	}
	if o.LocalLogDir != "" {
		args = append(args, "-L"+o.LocalLogDir)
	}
	if o.RemoteLogDir != "" {
		args = append(args, "-R"+o.RemoteLogDir)
	}
	if o.FileChecksum != NONE_CHECKSUM {
		if s, err := o.FileChecksum.getCommandLineString(); err == nil {
			args = append(args, "--file-checksum="+s)			
		}
	}
	jsonOptions, err := json.Marshal(asperaJsonOptions{
		Inner: innerAsperaJsonOptions{
			XferId:    o.getTransferIdOrSetDefault(),
			XferRetry: o.TransferRetry,
		},
	})
	if err != nil {
		fmLog.Printf("TransferOptions: Error encoding JSON tags: %v\n", err)
	} else {
		optsStr := base64.StdEncoding.EncodeToString(jsonOptions)
		args = append(args, "--tags64")
		args = append(args, optsStr) // It's important that this be added separately from '--tags64'
	}
	if len(o.ExcludePatterns) > 0 {
		for _, v := range o.ExcludePatterns {
			args = append(args, "-E"+v)
		}
	}
	if p, err := o.OverwritePolicy.getCommandLineString(); err == nil {
		args = append(args, "--overwrite="+p)
	}
	if o.SourceBase != "" {
		args = append(args, "--src-base="+o.SourceBase)
	}
	if o.ReadSize > 0 {
		args = append(args, "-g "+strconv.Itoa(o.ReadSize))
	}
	if o.WriteSize > 0 {
		args = append(args, "-G "+strconv.Itoa(o.WriteSize))
	}
	if o.RemoveFileAfterTransfer ||
		o.DeleteSource == FILES_AND_DIRECTORIES_DELETE_SOURCE ||
		o.DeleteSource == FILES_ONLY_DELETE_SOURCE {

		args = append(args, "--remove-after-transfer")
	}
	if o.RemoveEmptyDirectories ||
		o.DeleteSource == EMPTY_DIRECTORIES_DELETE_SOURCE ||
		o.DeleteSource == FILES_AND_DIRECTORIES_DELETE_SOURCE {

		args = append(args, "--remove-empty-directories")
	}
	if o.PreCalculateJobSize {
		args = append(args, "--precalculate-job-size")
	}
	if o.PrePostCommandPath != "" {
		args = append(args, "-e "+o.PrePostCommandPath)
	}
	if o.AlternateConfigFileName != "" {
		args = append(args, "-f "+o.AlternateConfigFileName)
	}
	if o.PartialFileSuffix != "" {
		args = append(args, "--partial-file-suffix="+o.PartialFileSuffix)
	}
	if o.ExcludeNewerThan != 0 {
		args = append(args, "--exclude-newer-than="+strconv.Itoa(o.ExcludeNewerThan))
	}
	if o.ExcludeOlderThan != 0 {
		args = append(args, "--exclude-older-than="+strconv.Itoa(o.ExcludeOlderThan))
	}
	if o.SaveBeforeOverwriteEnabled {
		args = append(args, "--save-before-overwrite")
	}
	if o.CheckSshFingerprint != "" {
		args = append(args, "--check-sshfp="+o.CheckSshFingerprint)
	}
	if o.RetransmissionRequestMaxSize > 0 {
		args = append(args, "-X"+strconv.Itoa(o.RetransmissionRequestMaxSize))
	}
	if o.HttpFallback != nil {
		args = append(args, o.HttpFallback.getCommandLineStrings()...)
	}
	if o.MoveAfterTransferPath != "" {
		args = append(args, "--move-after-transfer="+o.MoveAfterTransferPath)
	}

	// TODO: Remove this check when ASCP 4 supports these.
	if !o.ascp4StreamFlag {
		if presXAttrs, err := o.PreserveXAttrs.getCommandLineString(); err == nil {
			args = append(args, "--preserve-xattrs="+presXAttrs)
		}
		if remotePresXAttrs, err := o.RemotePreserveXAttrs.getCommandLineString(); err == nil {
			args = append(args, "--remote-preserve-xattrs="+remotePresXAttrs)
		}
		if presAcls, err := o.PreserveAcls.getCommandLineString(); err == nil {
			args = append(args, "--preserve-acls="+presAcls)
		}
		if remotePresAcls, err := o.RemotePreserveAcls.getCommandLineString(); err == nil {
			args = append(args, "--remote-preserve-acls="+remotePresAcls)
		}
	}

	if o.DeleteBeforeTransfer {
		args = append(args, "--delete-before-transfer")
	}
	if o.PreserveCreationTime {
		args = append(args, "--preserve-creation-time")
	}
	if o.PreserveModificationTime {
		args = append(args, "--preserve-modification-time")
	}
	if o.PreserveAccessTime {
		args = append(args, "--preserve-access-time")
	}
	if o.PreserveSourceAccessTime {
		args = append(args, "--preserve-source-access-time")
	}
	if o.SkipDirTraversalDupes {
		args = append(args, "--skip-dir-traversal-dupes")
	}
	if o.ApplyLocalDocroot {
		args = append(args, "--apply-local-docroot")
	}
	if o.MultiSessionThreshold != 0 {
		args = append(args, "--multi-session-threshold="+strconv.Itoa(o.MultiSessionThreshold))
	}
	if o.IgnoreHostKey {
		args = append(args, "--ignore-host-key")
	}
	if len(o.SourcePrefix) > 0 {
		args = append(args, "--source-prefix="+o.SourcePrefix)
	}
	if o.RemoveEmptySourceDirectory {
		args = append(args, "--remove-empty-source-directory")
	}
	for _, option := range o.ExtraOptions {
		args = append(args, option)
	}

	return args
}

func (o *TransferOptions) validate() error {
	if o.DatagramSize != 0 && (o.DatagramSize < 296 || o.DatagramSize > 10000) {
		return errors.New("TransferOptions: If specified, DatagramSize must be between " +
			"296 and 10000. Was: " + strconv.Itoa(o.DatagramSize))
	}
	if o.MinRateKbps < 0 {
		return errors.New("TransferOptions: If specified, MinRateKbps must be a non-negative int. " +
			"Was: " + strconv.Itoa(o.MinRateKbps))
	}
	if o.TargetRateKbps < 0 {
		return errors.New("TransferOptions: If specified, TargetRateKbps must be a non-negative int. " +
			"Was: " + strconv.Itoa(o.TargetRateKbps))
	}
	if o.TargetRateKbps < o.MinRateKbps {
		return errors.New("TransferOptions: MinRateKbps must be less than or equal to TargetRateKbps. " +
			"MinRateKbps was: " + strconv.Itoa(o.MinRateKbps) + ". TargetRateKbps was: " + strconv.Itoa(o.TargetRateKbps))
	}
	if o.TcpPort < 0 || o.TcpPort > 65535 {
		return errors.New("TransferOptions: If specified, TcpPort must be a positive int in the range [0, 65535]. " +
			"Was: " + strconv.Itoa(o.TcpPort))
	}
	if o.UdpPort < 0 || o.UdpPort > 65535 {
		return errors.New("TransferOptions: If specified, UdpPort must be a positive int in the range [0, 65535]." +
			"Was: " + strconv.Itoa(o.UdpPort))
	}
	if o.AutoDetectCapacity && (o.MinRateKbps > 100 || o.TargetRateKbps > 100) {
		return errors.New("TransferOptions: When AutoDetectCapacity is true, MinRateKbps and TargetRateKbps " +
			"are interpreted as percentages of measured bandwidth and must be less than or equal to 100. " +
			"MinRateKbps was: " + strconv.Itoa(o.MinRateKbps) + ". TargetRateKbps was: " + strconv.Itoa(o.TargetRateKbps))
	}
	if o.RetryTimeoutS < 0 {
		return errors.New("TransferOptions: When specified, RetryTimeoutS must be positive. " +
			"Was: " + strconv.Itoa(o.RetryTimeoutS))
	}
	if len(o.ExcludePatterns) > 16 {
		return errors.New("TransferOptions: Maximum of 16 ExcludePatterns allowed. " +
			"Was: " + strconv.Itoa(len(o.ExcludePatterns)))
	}
	if o.TransferRetry < 0 {
		return errors.New("TransferOptions: If specified, TransferRetry must be a positive number. Was " +
			strconv.Itoa(o.TransferRetry))
	}
	if o.HttpFallback != nil {
		if err := o.HttpFallback.validate(); err != nil {
			return err
		}
		if o.DeleteBeforeTransfer {
			return errors.New("TransferOptions: Options DeleteBeforeTransfer and HttpFallback may not be used together.")
		}
	}
	if len(o.MoveAfterTransferPath) > 0 && o.RemoveFileAfterTransfer {
		return errors.New("TransferOptions: May not set both RemoveFileAfterTransfer and MoveAfterTransferPath.")
	}
	if runtime.GOOS == "windows" && o.SymLinkPolicy != UNINITIALIZED_SYM_LINK_POLICY {
		return errors.New("TransferOptions: SymLinkPolicy not supported on Windows.")
	}

	// TODO: Remove this check when ASCP 4 supports all options.
	if o.ascp4StreamFlag {
		return o.validateForAscp4()
	}

	return nil
}

func (o *TransferOptions) validateForAscp4() error {
	if o.ExcludePatterns != nil {
		return errors.New("TransferOptions: ASCP4 streaming doesn't support exclude patterns.")
	}
	if o.ExcludeOlderThan != 0 {
		return errors.New("TransferOptions: ASCP4 streaming doesn't support ExcludeOlderThan.")
	}
	if o.ExcludeNewerThan != 0 {
		return errors.New("TransferOptions: ASCP4 streaming doesn't support ExcludeNewerThan.")
	}
	if o.SaveBeforeOverwriteEnabled {
		return errors.New("TransferOptions: ASCP4 streaming doesn't support SaveBeforeOverwriteEnabled")
	}
	if o.PreserveCreationTime {
		return errors.New("TransferOptions: ASCP4 streaming doesn't support PreserveCreationTime.")
	}
	if o.SkipDirTraversalDupes {
		return errors.New("TransferOptions: ASCP4 streaming doesn't support SkipDirTraversalDupes.")
	}
	if o.ApplyLocalDocroot {
		return errors.New("TransferOptions: ASCP4 streaming doesn't support ApplyLocalDocroot.")
	}
	if o.MultiSessionThreshold != 0 {
		return errors.New("TransferOptions: ASCP4 streaming doesn't support MultiSessionThreshold.")
	}
	if o.RemoveEmptySourceDirectory {
		return errors.New("TransferOptions: ASCP4 streaming doesn't support RemoveEmptySourceDirectory.")
	}
	if len(o.SourcePrefix) > 0 {
		return errors.New("TransferOptions: ASCP4 streaming doesn't support SourcePrefixes.")
	}
	if o.DeleteBeforeTransfer {
		return errors.New("TransferOptions: ASCP4 streaming doesn't support DeleteBeforeTransfer.")
	}
	if len(o.MoveAfterTransferPath) > 0 {
		return errors.New("TransferOptions: ASCP4 streaming doesn't support RemoveAfterTransferPath.")
	}
	if len(o.CheckSshFingerprint) > 0 {
		return errors.New("TransferOptions: ASCP4 streaming doesn't support CheckSshFingerprint.")
	}
	if o.PreserveXAttrs != NONE_PRESERVE_MODE ||
		o.RemotePreserveXAttrs != NONE_PRESERVE_MODE {
		return errors.New("TransferOptions: ASCP4 streaming doesn't support PreserveXAttrs.")
	}
	if o.PreserveAcls != NONE_PRESERVE_MODE ||
		o.RemotePreserveAcls != NONE_PRESERVE_MODE {
		return errors.New("TransferOptions: ASCP4 streaming doesn't support PreserveAcls.")
	}
	if len(o.FileManifestPath) != 0 {
		return errors.New("TransferOptions: ASCP4 streaming doesn't support FileManifestPath.")
	}
	return nil
}
