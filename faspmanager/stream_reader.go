package faspmanager

// StreamReader is the building block of a stream-to-file transfer. It represents a
// readable stream of bytes and information about the offset at which the bytes
// should be written in some remote file.
type StreamReader interface {

	// Read reads from an underlying input source and returns a StreamReadBlock, a
	// struct containing both the read data and an offset in the remote destination
	// to which the bytes should be written. It should return an io.EOF error when
	// the StreamReader is consumed.
	Read() (StreamReadBlock, error)
}

// StreamReadBlock is the result of a call to StreamReader.Read.
type StreamReadBlock struct {

	// Buffer contains the bytes read with a call to StreamReader.Read.
	Buffer []byte

	// Position is the 0-based offset in the remote destination at which the bytes
	// in StreamReadBlock.Buffer are to be written. It must be greater than or
	// equal to zero and less than the size of the remote file (in bytes) or equal
	// to -1, which specifies a write to the end of the destination file.
	Position int
}

type StreamErr struct {
	StreamOpenErr  error // Error initiating stream with remote host.
	SourceReadErr  error // Error reading from stream source.
	StreamSendErr  error // Error sending data over stream.
	StreamCloseErr error // Error closing stream.
}

func (s StreamErr) Error() string {
	var msg string = "StreamErr: "
	if s.StreamOpenErr != nil {
		msg += "\n\tError opening stream: " + s.StreamOpenErr.Error()
	}
	if s.SourceReadErr != nil {
		msg += "\n\tError reading from reader: " + s.SourceReadErr.Error()
	}
	if s.StreamSendErr != nil {
		msg += "\n\tError sending data: " + s.StreamSendErr.Error()
	}
	if s.StreamCloseErr != nil {
		msg += "\n\tError closing stream: " + s.StreamCloseErr.Error()
	}
	return msg
}
