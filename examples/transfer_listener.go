package examples

import (
	"fmt"
	"path"

	"github.com/pacgenesis/faspmanager-sdk-go/faspmanager"
)

type TransferListener struct{}

func (l *TransferListener) OnTransferEvent(event faspmanager.TransferEvent) {
	switch event.EventType {
	case faspmanager.CONNECTING_TRANSFER_EVENT:
		fmt.Printf("CONNECTING EVENT: Session is connecting...\n\n")
	case faspmanager.SESSION_START_TRANSFER_EVENT:
		fmt.Printf("SESSION_START_EVENT: Started at: %v\n\n", event.SessionStats.StartTime)
	case faspmanager.RATE_MODIFICATION_TRANSFER_EVENT:
		fmt.Printf("RATE_MODIFICATION_EVENT: Rate modification. \n\n\tTarget Rate: %d KB/s\n\n\tMin Rate: %dKB/s\n\n",
			event.SessionStats.TargetRateKbps, event.SessionStats.MinRateKbps)
	case faspmanager.FILE_START_TRANSFER_EVENT:
		if event.FileStats != nil {
			fmt.Printf("FILE_START_EVENT: %v transfer has started.\n\n", path.Base(event.FileStats.Name))
		}
	case faspmanager.PROGRESS_TRANSFER_EVENT:
		if event.FileStats != nil {
			fmt.Printf("PROGRESS_EVENT: Written %d bytes for %s\n\n",
				event.FileStats.WrittenBytes, path.Base(event.FileStats.Name))
		}
	case faspmanager.FILE_STOP_TRANSFER_EVENT:
		if event.FileStats != nil {
			fmt.Printf("FILE_STOP_EVENT: %v transfer has completed.\n\n", path.Base(event.FileStats.Name))
		}
	case faspmanager.FILE_ERROR_TRANSFER_EVENT:
		fmt.Printf("FILE_ERROR_EVENT: %v\n\n", event.FileStats.ErrorDescription)
	case faspmanager.SESSION_ERROR_TRANSFER_EVENT:
		fmt.Printf("SESSION_ERROR_EVENT: %v\n\n", event.SessionStats.ErrorDescription)
	case faspmanager.SESSION_STOP_TRANSFER_EVENT:
		fmt.Println("SESSION_STOP_EVENT: Statistics:\n")
		fmt.Printf("\tFiles complete: %d\n", event.SessionStats.FilesComplete)
		fmt.Printf("\tFiles failed: %d\n", event.SessionStats.FilesFailed)
		fmt.Printf("\tFiles skipped: %d\n", event.SessionStats.FilesSkipped)
		fmt.Printf("\tTotal written bytes: %d\n", event.SessionStats.TotalWrittenBytes)
		fmt.Printf("\tTotal transferred bytes: %d\n", event.SessionStats.TotalTransferredBytes)
		fmt.Printf("\tTotal lost bytes: %d\n\n", event.SessionStats.TotalLostBytes)
	case faspmanager.FILE_SKIP_TRANSFER_EVENT:
		if event.FileStats != nil {
			fmt.Printf("FILE_SKIP_EVENT: %v skipped.\n\n", path.Base(event.FileStats.Name))
		}
	}
}
