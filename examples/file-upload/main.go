package main

import (
	"math/rand"
	"strconv"
	
	"github.com/pacgenesis/faspmanager-sdk-go/examples"
	"github.com/pacgenesis/faspmanager-sdk-go/faspmanager"
)

func main() {

	manager, err := faspmanager.New()
	if err != nil {
		panic(err)
	}
	defer manager.Close()

	err = manager.SetAscpPath(examples.ASCP)
	if err != nil {
		panic(err)
	}

	transferOrder := faspmanager.FileUpload{
		DestHost: examples.REMOTE_HOST,
		DestUser: examples.REMOTE_USER,
		DestPass: examples.REMOTE_PASS,
		SourcePath: examples.MEDIUM_LOCAL_FILE,
		DestPath: examples.REMOTE_DIR+"/"+strconv.Itoa(rand.Int()),
	}

	listener := &examples.TransferListener{}

	transferId, err := manager.StartTransferWithListener(transferOrder, listener)
	if err != nil {
		panic(err)
	}

	err = manager.WaitOnSessionStop(transferId)
	if err != nil {
		panic(err)
	}

}
