package faspmanager

import (
	"io"
	"net"
	"strconv"
	"sync"
)

type streamJob interface{}

type readerJob struct {
	DestPath string
	Source   io.Reader
}

type streamReaderJob struct {
	DestPath string
	Source   StreamReader
}

type streamWorker struct {
	mgmtConn      net.Conn
	mgmtWriteSync *sync.Mutex
	chunkSize     int
	jobQ          chan streamJob
	complete      chan struct{}
	terminate     chan struct{}
	onStreamErr   func(err StreamErr, destPath string)
}

func newStreamWorker(mgmtConn net.Conn,
	mgmtWriteSync *sync.Mutex,
	chunkSize int,
	complete chan struct{},
	terminate chan struct{},
	onStreamErr func(err StreamErr, destPath string)) *streamWorker {
	return &streamWorker{
		mgmtConn:      mgmtConn,
		mgmtWriteSync: mgmtWriteSync,
		chunkSize:     chunkSize,
		jobQ:          make(chan streamJob, 32),
		complete:      complete,
		terminate:     terminate,
		onStreamErr:   onStreamErr,
	}
}

func (w *streamWorker) start() {
	go func() {
		for job := range w.jobQ {
			select {
			case <-w.terminate:
				return
			case <-w.complete:
				return
			default:
			}

			switch job.(type) {
			case readerJob:
				w.sendStream(job.(readerJob))
			case streamReaderJob:
				w.sendStreamReader(job.(streamReaderJob))
			}
		}
	}()
}

func (w *streamWorker) enqueue(job streamJob) {
	go func() {
		select {
		case <-w.terminate:
		case <-w.complete:
		case w.jobQ <- job:
		}
	}()
}

func (w *streamWorker) sendStream(job readerJob) {
	err := w.sendPut(job.DestPath)
	if err != nil {
		streamErr := StreamErr{StreamOpenErr: err}
		w.onStreamErr(streamErr, job.DestPath)
		return
	}

	var readErr, sendErr, closeErr error
	for n, bytesRead, currOffset, buff := 0, 0, 0, make([]byte, w.chunkSize); readErr == nil && sendErr == nil; {
		select {
		case <-w.terminate:
			break
		default:
		}

		n, readErr = job.Source.Read(buff[bytesRead:])
		bytesRead += n
		if bytesRead == len(buff) || bytesRead > 0 && readErr != nil {
			sendErr = w.sendWrite(job.DestPath, currOffset, bytesRead, buff[:bytesRead])
			currOffset += bytesRead
			bytesRead = 0
		}
	}

	closeErr = w.sendClose(job.DestPath)
	if (readErr != nil && readErr != io.EOF) || sendErr != nil || closeErr != nil {
		streamErr := StreamErr{
			SourceReadErr:  readErr,
			StreamSendErr:  sendErr,
			StreamCloseErr: closeErr,
		}
		w.onStreamErr(streamErr, job.DestPath)
	}
}

func (w *streamWorker) sendStreamReader(job streamReaderJob) {
	err := w.sendPut(job.DestPath)
	if err != nil {
		streamErr := StreamErr{StreamOpenErr: err}
		w.onStreamErr(streamErr, job.DestPath)
		return
	}

	var readBlock StreamReadBlock
	var readErr, sendErr, closeErr error
	for numBytesWritten := 0; readErr == nil && sendErr == nil; {
		select {
		case <-w.terminate:
			break
		default:
		}

		readBlock, readErr = job.Source.Read()
		if len(readBlock.Buffer) == 0 && readErr != nil {
			break
		}
		if readBlock.Position == -1 {
			readBlock.Position = numBytesWritten
		}
		readErr = w.sendWrite(job.DestPath, readBlock.Position, len(readBlock.Buffer), readBlock.Buffer)
		numBytesWritten += len(readBlock.Buffer)
	}

	closeErr = w.sendClose(job.DestPath)
	if (readErr != nil && readErr != io.EOF) || sendErr != nil || closeErr != nil {
		streamErr := StreamErr{
			SourceReadErr:  readErr,
			StreamSendErr:  sendErr,
			StreamCloseErr: closeErr,
		}
		w.onStreamErr(streamErr, job.DestPath)
	}
}

func (w *streamWorker) sendPut(destPath string) error {
	msg := "FASPMGR 4\nType: PUT\nDestination: " + destPath + "\n\n"
	w.mgmtWriteSync.Lock()
	err := writeAll(w.mgmtConn, []byte(msg))
	w.mgmtWriteSync.Unlock()
	return err
}

func (w *streamWorker) sendWrite(destPath string, offset int, size int, payload []byte) error {
	header := "FASPMGR 4\nType: WRITE\nDestination: " + destPath +
		"\nOffset: " + strconv.Itoa(offset) +
		"\nSize: " + strconv.Itoa(size) + "\n\n"
	w.mgmtWriteSync.Lock()
	defer w.mgmtWriteSync.Unlock()
	err := writeAll(w.mgmtConn, []byte(header))
	if err != nil {
		return err
	}
	err = writeAll(w.mgmtConn, payload)
	if err != nil {
		return err
	}
	err = writeAll(w.mgmtConn, []byte("\n\n"))
	if err != nil {
		return err
	}
	return nil
}

func (w *streamWorker) sendClose(destPath string) error {
	msg := "FASPMGR 4\nType: CLOSE\nDestination: " + destPath + "\n\n"
	w.mgmtWriteSync.Lock()
	err := writeAll(w.mgmtConn, []byte(msg))
	w.mgmtWriteSync.Unlock()
	return err
}

func writeAll(conn net.Conn, payload []byte) error {
	for written := 0; written < len(payload); {
		n, err := conn.Write(payload[written:])
		if err != nil {
			return err
		}
		written += n
	}
	return nil
}
