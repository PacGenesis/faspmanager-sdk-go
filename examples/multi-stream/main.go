package main

import (
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"sync"

	"github.com/pacgenesis/faspmanager-sdk-go/examples"
	"github.com/pacgenesis/faspmanager-sdk-go/faspmanager"
)

const (
	numFiles = 20
	fileSize = 1000000
)

func getFiles() ([]*os.File, error) {
	var res []*os.File
	for i := 0; i < numFiles; i++ {
		file, err := ioutil.TempFile("", "")
		if err != nil {
			return nil, err
		}
		err = os.Truncate(file.Name(), fileSize)
		if err != nil {
			return nil, err
		}
		res = append(res, file)
	}
	return res, nil
}

type TransferState struct {
	Abort   chan struct{}
	Ongoing map[string]chan struct{}
	sync.Mutex
}

func (state *TransferState) OnTransferEvent(event faspmanager.TransferEvent) {
	switch event.EventType {
	case faspmanager.SESSION_ERROR_TRANSFER_EVENT:

		log.Printf("Received session error. Code: %d. Description: %s.\n",
			event.SessionStats.ErrorCode, event.SessionStats.ErrorDescription)
		close(state.Abort)

	case faspmanager.FILE_STOP_TRANSFER_EVENT,
		faspmanager.FILE_ERROR_TRANSFER_EVENT:

		if event.EventType == faspmanager.FILE_ERROR_TRANSFER_EVENT {
			log.Printf("Received file error. Filename: %s, Code: %d, Description: %s\n",
				event.FileStats.Name, event.FileStats.ErrorCode, event.FileStats.ErrorDescription)
		}

		state.Lock()
		fileComplete, ok := state.Ongoing[event.FileStats.Name]
		state.Unlock()
		if !ok {
			log.Printf("Received update for unknown file. Filename: %s\n.", event.FileStats.Name)
			return
		}

		close(fileComplete)
	}
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

	transferState := &TransferState{Abort: make(chan struct{}), Ongoing: make(map[string]chan struct{})}

	manager.AddGlobalListener(transferState)

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

	sessId, err := manager.StartTransfer(order)
	if err != nil {
		panic(err)
	}

	err = manager.WaitOnSessionStart(sessId)
	if err != nil {
		panic(err)
	}

	files, err := getFiles()
	if err != nil {
		panic(err)
	}

	var wg sync.WaitGroup

	log.Printf("Beginning transfer of %d files.\n", len(files))
	for i, file := range files {
		select {
		case <-transferState.Abort:
			log.Println("Session failure. Exiting.")
			os.Exit(1)
		default:
		}

		wg.Add(1)
		go func(i int, file *os.File) {
			defer wg.Done()

			fileComplete := make(chan struct{})

			destinationPath := "file" + strconv.Itoa(i)

			transferState.Lock()
			transferState.Ongoing[destinationPath] = fileComplete
			transferState.Unlock()

			err := manager.AddReaderSource(sessId, file, destinationPath)
			if err != nil {
				log.Printf("Unable to add %s. Error: %s.\n", destinationPath, err.Error())
				return
			}

			<-fileComplete

			log.Printf("%s complete.\n", destinationPath)
		}(i, file)

	}

	wg.Wait()
}
