package main

import (
	"io"

	"github.com/pacgenesis/faspmanager-sdk-go/examples"
	"github.com/pacgenesis/faspmanager-sdk-go/faspmanager"
)

var BYTES_TO_WRITE = 128 * 1024 * 3

const (
	DESTINATION_PATH = "AAA_STREAM_TEST_FILE"
)

type MyReader struct {
	bytesWritten int
}

func (m *MyReader) Read(b []byte) (n int, err error) {
	if m.bytesWritten >= BYTES_TO_WRITE {
		return 0, io.EOF
	}

	var i int
	for i < len(b) && m.bytesWritten < BYTES_TO_WRITE {
		b[i] = 'x'
		m.bytesWritten++
		i++
	}

	return i, nil
}

func main() {

	manager, err := faspmanager.New()
	if err != nil {
		panic(err)
	}
	defer manager.Close()

	err = manager.SetAscpPath(examples.ASCP4)
	if err != nil {
		panic(err)
	}

	order := faspmanager.PersistentStreamUpload{
		DestHost: examples.REMOTE_HOST,
		DestUser: examples.REMOTE_USER,
		DestPass: examples.REMOTE_PASS,
		Options: &faspmanager.TransferOptions{
			ChunkSize: 64 * 1024, // Ensure matching server aspera.conf <chunk_size>65536</chunk_size>
			DestinationRoot: "DestinationRoot",
			TargetRateKbps: 100000,
		},

	}

	id, err := manager.StartTransferWithListener(order, &examples.TransferListener{})
	if err != nil {
		panic(err)
	}

	err = manager.WaitOnSessionStart(id)
	if err != nil {
		panic(err)
	}

	err = manager.AddReaderSource(id, &MyReader{}, DESTINATION_PATH)
	if err != nil {
		panic(err)
	}

	err = manager.LockPersistentSession(id)
	if err != nil {
		panic(err)
	}

	err = manager.WaitOnSessionStop(id)
	if err != nil {
		panic(err)
	}
}
