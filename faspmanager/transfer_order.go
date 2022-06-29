package faspmanager

import (
	"bufio"
	"errors"
	"io/ioutil"
	"net"
	"strconv"
)

// TransferOrder wraps the GetAscpOptions method, which returns an AscpOptions
// struct containing information needed to start a FASP transfer.

// The interface exists to provide convenient syntax for requesting transfers, and
// isn't intended to be implemented directly.  Instead, use one of the provided
// types: FileUpload, FileDownload, MultiFileUpload, MultiFileDownload,
// PersistentUpload, PersistentDownload, and PersistentStreamUpload.
type TransferOrder interface {
	GetAscpOptions() (*AscpOptions, error)
}

type DirectionType int

const (
	UploadDirection DirectionType = iota
	DownloadDirection
)

// AscpOptions encloses all information about a requested transfer.
type AscpOptions struct {
	SourcePaths    []string
	DestPaths      []string
	Direction      DirectionType
	RemoteHost     string
	RemoteUser     string
	RemotePass     string
	RemoteIdentity string
	Persistent     bool
	Stream         bool
	TransferOpts   *TransferOptions
}

func (o *AscpOptions) buildAscpCommandLine() ([]string, error) {
	if o.TransferOpts == nil {
		o.TransferOpts = &TransferOptions{}
	}

	// TODO: Remove this when ASCP 4 supports all options.
	if o.Stream {
		o.TransferOpts.ascp4StreamFlag = true
	}

	if err := o.TransferOpts.validate(); err != nil {
		return nil, err
	}

	var cmd []string

	if o.Stream {
		cmd = append(cmd, "--faspmgr-io")
	} else if o.Persistent {
		cmd = append(cmd, "--keepalive")
	}

	// Check if the remote host is an IPv6 address.
	if ip := net.ParseIP(o.RemoteHost); ip != nil && ip.To4() == nil {
		cmd = append(cmd, "-6")
	}
	if o.Direction == DownloadDirection {
		cmd = append(cmd, "--mode=recv")
	} else {
		cmd = append(cmd, "--mode=send")
	}
	cmd = append(cmd, "-q")
	cmd = append(cmd, o.TransferOpts.getCommandLineParams()...)
	if len(o.TransferOpts.ContentProtectionPassphrase) > 0 {
		if o.Direction == DownloadDirection {
			cmd = append(cmd, "--file-crypt=decrypt")
		} else {
			cmd = append(cmd, "--file-crypt=encrypt")
		}
	}

	cmd = append(cmd, "--host="+o.RemoteHost)
	cmd = append(cmd, "--user="+o.RemoteUser)
	if len(o.RemoteIdentity) > 0 {
		cmd = append(cmd, "-i")
		cmd = append(cmd, o.RemoteIdentity)
	}

	// Add the source and destination information, if sources are present.
	if len(o.SourcePaths) > 0 {
		sourceFileName, err := writeTransferSourcesFile(o.SourcePaths, o.DestPaths)
		if err != nil {
			return nil, err
		}
		if len(o.SourcePaths) > 1 && (len(o.SourcePaths) == len(o.DestPaths)) {
			cmd = append(cmd, "--file-pair-list="+sourceFileName)
		} else {
			cmd = append(cmd, "--file-list="+sourceFileName)
		}
	}

	if o.Stream {
		cmd = append(cmd, "--chunk-size="+strconv.Itoa(o.TransferOpts.getChunkSizeOrSetDefault()))
	}

	// Add the destination root directory.
	if len(o.TransferOpts.DestinationRoot) > 0 {
		cmd = append(cmd, o.TransferOpts.DestinationRoot)
	} else if len(o.DestPaths) > 0 {
		cmd = append(cmd, o.DestPaths[0])
	} else {
		cmd = append(cmd, DefaultDestRoot)
	}

	return cmd, nil
}

// Writes a temporary file containing either a list of matching source/dest pairs
// (if len(sourcePaths) > 1 && len(sourcePaths) == len(destPaths)) or just of
// source paths otherwise.
func writeTransferSourcesFile(sourcePaths []string, destPaths []string) (string, error) {
	f, err := ioutil.TempFile("", "faspmanagerFileList")
	if err != nil {
		return "", err
	}
	defer f.Close()
	writer := bufio.NewWriter(f)
	if len(sourcePaths) > 1 && (len(sourcePaths) == len(destPaths)) {
		for i, path := range sourcePaths {
			writer.WriteString(path + "\n")
			writer.WriteString(destPaths[i] + "\n")
		}
	} else {
		for _, path := range sourcePaths {
			writer.WriteString(path + "\n")
		}
	}
	writer.Flush()
	return f.Name(), nil
}

// TransferOrder implementations.

// FileUpload is a single file upload.
type FileUpload struct {

	// The location of the source.
	SourcePath string

	// The destination location.
	DestPath string

	// The remote login username.
	DestUser string

	// The remote host. (e.g. "localhost" or "10.0.0.2")
	DestHost string

	// The login password or passphrase associated with the identity file (private
	// key).
	DestPass string

	// The path to the identity file (private key) for use in public key
	// authentication.  This may be left empty if not needed.
	DestIdentity string

	// Transfer options. These may be left nil.
	Options *TransferOptions
}

func (o FileUpload) GetAscpOptions() (*AscpOptions, error) {
	if len(o.SourcePath) == 0 || len(o.DestPath) == 0 || len(o.DestUser) == 0 ||
		len(o.DestHost) == 0 || (len(o.DestPass) == 0 && len(o.DestIdentity) == 0) {
		return nil, errors.New("FileUpload: Must provide SourcePath, DestPath, DestHost, DestUser, and DestPass or DestIdentity")
	}

	return &AscpOptions{
		SourcePaths:    []string{o.SourcePath},
		DestPaths:      []string{o.DestPath},
		RemoteHost:     o.DestHost,
		RemoteUser:     o.DestUser,
		RemotePass:     o.DestPass,
		RemoteIdentity: o.DestIdentity,
		TransferOpts:   o.Options,
		Persistent:     false,
		Direction:      UploadDirection,
	}, nil
}

// FileDownload is a single file download.
type FileDownload struct {

	// The location of the source.
	SourcePath string

	// The destination location.
	DestPath string

	// The remote login username.
	SourceUser string

	// The remote host. (e.g. "localhost" or "10.0.0.2")
	SourceHost string

	// The login password or passphrase associated with the source.
	SourcePass string

	// The path to the identity file (private key) for use in public key authentication.
	// This may be left empty if not needed.
	SourceIdentity string

	// Transfer options. These may be left nil.
	Options *TransferOptions
}

func (o FileDownload) GetAscpOptions() (*AscpOptions, error) {
	if len(o.SourcePath) == 0 || len(o.DestPath) == 0 || len(o.SourceUser) == 0 || len(o.SourceHost) == 0 ||
		(len(o.SourcePass) == 0 && len(o.SourceIdentity) == 0) {
		return nil, errors.New("FileDownload: Must provide SourcePath, DestPath, SourceHost, SourceUser, and SourcePass or SourceIdentity")
	}

	return &AscpOptions{
		SourcePaths:    []string{o.SourcePath},
		DestPaths:      []string{o.DestPath},
		RemoteHost:     o.SourceHost,
		RemoteUser:     o.SourceUser,
		RemotePass:     o.SourcePass,
		RemoteIdentity: o.SourceIdentity,
		TransferOpts:   o.Options,
		Persistent:     false,
		Direction:      DownloadDirection,
	}, nil
}

// MultiFileUpload is a transfer of multiple sources to either a single destination
// or a destination for each source.
type MultiFileUpload struct {

	// The locations of the source files.
	SourcePaths []string

	// The path of the destination or destinations.  May consist of either a single
	// destination path (e.g. []string { /my/dest/path.txt }) or a number of
	// destination paths matching the number of source paths. Destination
	// directories will be matched by index with source files.
	DestPaths []string

	// The remote login username.
	DestUser string

	// The remote host. (e.g. "localhost" or "10.0.0.2")
	DestHost string

	// The login password or passphrase associated with the destination.
	DestPass string

	// The path to the identity file (private key) for use in public key
	// authentication.  This may be left empty if not needed.
	DestIdentity string

	// Transfer options. These may be left nil.
	Options *TransferOptions
}

func (o MultiFileUpload) GetAscpOptions() (*AscpOptions, error) {
	if len(o.SourcePaths) == 0 || len(o.DestPaths) == 0 || len(o.DestHost) == 0 ||
		len(o.DestUser) == 0 || (len(o.DestPass) == 0 && len(o.DestIdentity) == 0) {
		return nil, errors.New("MultiFileUpload: Must provide SourcePaths, DestPaths, DestHost, DestUser, and DestPass or DestIdentity")
	}
	if len(o.DestPaths) > 1 && len(o.SourcePaths) != len(o.DestPaths) {
		return nil, errors.New("MultiFileUpload: Must either provide a single destination path or a number " +
			"of destination paths equal to the number of source paths.")
	}

	return &AscpOptions{
		SourcePaths:    o.SourcePaths,
		DestPaths:      o.DestPaths,
		RemoteHost:     o.DestHost,
		RemoteUser:     o.DestUser,
		RemotePass:     o.DestPass,
		RemoteIdentity: o.DestIdentity,
		TransferOpts:   o.Options,
		Persistent:     false,
		Direction:      UploadDirection,
	}, nil
}

// MultiFileDownload is a download of multiple sources to either a single
// destination or a destination for each source.
type MultiFileDownload struct {

	// The locations of the source files.
	SourcePaths []string

	// The path of the destination directory or directories.  It may consist of
	// either a single destination path (e.g. []string { /my/dest/path }) or a
	// number of destination paths matching the number of source
	// paths. Destinations will be matched by index with sources.
	DestPaths []string

	// The remote login username.
	SourceUser string

	// The remote host. (e.g. "localhost" or "10.0.0.2")
	SourceHost string

	// The login password or passphrase associated with the source.
	SourcePass string

	// The path to the identity file (private key) for use in public key authentication.
	// This may be left empty if not needed.
	SourceIdentity string

	// Transfer options. These may be left nil.
	Options *TransferOptions
}

func (o MultiFileDownload) GetAscpOptions() (*AscpOptions, error) {
	if o.SourcePaths == nil || len(o.SourcePaths) == 0 || o.DestPaths == nil || len(o.DestPaths) == 0 ||
		len(o.SourceHost) == 0 || len(o.SourceUser) == 0 || (len(o.SourcePass) == 0 && len(o.SourceIdentity) == 0) {
		return nil, errors.New("MultiFileDownload: Must provide SourcePaths, DestPaths, SourceHost, SourceUser, and SourcePass or SourceIdentity")
	}
	if len(o.DestPaths) > 1 && len(o.SourcePaths) != len(o.DestPaths) {
		return nil, errors.New("MultiFileDownload: Must either provide a single destination path or a number " +
			"of destination paths equal to the number of source paths.")
	}

	return &AscpOptions{
		SourcePaths:    o.SourcePaths,
		DestPaths:      o.DestPaths,
		RemoteHost:     o.SourceHost,
		RemoteUser:     o.SourceUser,
		RemotePass:     o.SourcePass,
		RemoteIdentity: o.SourceIdentity,
		TransferOpts:   o.Options,
		Persistent:     false,
		Direction:      DownloadDirection,
	}, nil
}

// In a PersistentUpload, source and destination paths are not specified at the
// start of the session. Rather, they are added once the session is established
// with a call to one of Manager's AddSource methods and may continually be added
// until the session is closed.
type PersistentUpload struct {

	// The remote login username.
	DestUser string

	// The remote host. (e.g. "localhost" or "10.0.0.2")
	DestHost string

	// The login password or passphrase associated with the destination.
	DestPass string

	// The path to the identity file (private key) for use in public key
	// authentication.  This may be left empty if not needed.
	DestIdentity string

	// Transfer options. These may be left nil.
	Options *TransferOptions
}

func (o PersistentUpload) GetAscpOptions() (*AscpOptions, error) {
	if len(o.DestHost) == 0 || len(o.DestUser) == 0 || (len(o.DestPass) == 0 && len(o.DestIdentity) == 0) {
		return nil, errors.New("PersistentUpload: Must provide DestHost, DestUser, and DestPass or DestIdentity.")
	}

	return &AscpOptions{
		SourcePaths:    nil,
		DestPaths:      nil,
		RemoteHost:     o.DestHost,
		RemoteUser:     o.DestUser,
		RemotePass:     o.DestPass,
		RemoteIdentity: o.DestIdentity,
		TransferOpts:   o.Options,
		Persistent:     true,
		Direction:      UploadDirection,
	}, nil
}

// In a PersistentDownload, source and destination paths are not specified at the
// start of the transfer. Rather, they are added once the session is established
// with a call to one of Manager's AddSource methods and may continually be added
// until the session is locked.
type PersistentDownload struct {

	// The remote login username.
	SourceUser string

	// The remote host. (e.g. "localhost" or "10.0.0.2")
	SourceHost string

	// The login password or passphrase associated with the source.
	SourcePass string

	// The path to the identity file (private key) for use in public key
	// authentication.  This may be left empty if not needed.
	SourceIdentity string

	// Transfer options. These may be left nil.
	Options *TransferOptions
}

func (o PersistentDownload) GetAscpOptions() (*AscpOptions, error) {
	if len(o.SourceHost) == 0 || len(o.SourceUser) == 0 || len(o.SourcePass) == 0 ||
		(len(o.SourcePass) == 0 && len(o.SourceIdentity) == 0) {
		return nil, errors.New("PersistentDownload: Must provide SourceHost, SourceUser, and SourcePass or SourceIdentity.")
	}

	return &AscpOptions{
		SourcePaths:    nil,
		DestPaths:      nil,
		RemoteHost:     o.SourceHost,
		RemoteUser:     o.SourceUser,
		RemotePass:     o.SourcePass,
		RemoteIdentity: o.SourceIdentity,
		TransferOpts:   o.Options,
		Persistent:     true,
		Direction:      DownloadDirection,
	}, nil
}

// PersistentStreamUpload allows stream sources with Manager.AddReaderSource and
// Manager.AddStreamReaderSource.
type PersistentStreamUpload struct {

	// The remote login username.
	DestUser string

	// The remote host. (e.g. "localhost" or "10.0.0.2")
	DestHost string

	// The login password or passphrase associated with the identity file (private
	// key).
	DestPass string

	// The path to the identity file (private key) for use in public key
	// authentication.  This may be left empty if not needed.
	DestIdentity string

	// Transfer options. These may be left nil.
	Options *TransferOptions
}

func (o PersistentStreamUpload) GetAscpOptions() (*AscpOptions, error) {
	if len(o.DestHost) == 0 || len(o.DestUser) == 0 || (len(o.DestPass) == 0 && len(o.DestIdentity) == 0) {
		return nil, errors.New("PersistentStreamUpload: Must provide DestHost, DestUser, and DestPass or DestIdentity.")
	}

	return &AscpOptions{
		SourcePaths:    nil,
		DestPaths:      nil,
		RemoteHost:     o.DestHost,
		RemoteUser:     o.DestUser,
		RemotePass:     o.DestPass,
		RemoteIdentity: o.DestIdentity,
		TransferOpts:   o.Options,
		Stream:         true,
		Direction:      UploadDirection,
	}, nil
}
