package main

import (
	"math/rand"
	"strconv"

	"github.com/Aspera/faspmanager-sdk-go/examples"
	"github.com/Aspera/faspmanager-sdk-go/faspmanager"
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

	listener := &examples.TransferListener{}

	manager.AddGlobalListener(listener)

	transferOrder := faspmanager.PersistentUpload{
		DestHost: examples.REMOTE_HOST,
		DestUser: examples.REMOTE_USER,
		DestPass: examples.REMOTE_PASS,
		Options: &faspmanager.TransferOptions{
			TargetRateKbps:  25000,
		},
	}

	transferId, err := manager.StartTransfer(transferOrder)
	if err != nil {
		panic(err)
	}

	err = manager.WaitOnSessionStart(transferId)
	if err != nil {
		panic(err)
	}

	err = manager.AddFileSource(transferId, examples.SMALL_LOCAL_FILE, examples.REMOTE_DIR+"/persistent_upload"+strconv.Itoa(rand.Int()))
	if err != nil {
		panic(err)
	}

	err = manager.AddFileSource(transferId, examples.MEDIUM_LOCAL_FILE, examples.REMOTE_DIR+"/persistent_upload"+strconv.Itoa(rand.Int()))
	if err != nil {
		panic(err)
	}

	err = manager.LockPersistentSession(transferId)
	if err != nil {
		panic(err)
	}

	err = manager.WaitOnSessionStop(transferId)
	if err != nil {
		panic(err)
	}

}
