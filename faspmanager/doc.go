// Package faspmanager implements Aspera's FASP Manager API, providing
// facilities for programmatic initiation, monitoring // and management of FASP
// transfers.
//
// A simple example:
//
//        package main
//
//        import "github.com/pacgenesis/faspmanager-sdk-go/faspmanager"
//
//        func main() {
//                manager, _ := faspmanager.New()
//                defer manager.Close()
//                transferOrder := faspmanager.FileUpload{
//                      SourcePath: "/source-file.txt",
//                      DestPath:   "/destination-location.txt",
//                      DestHost:   "127.0.0.1",
//                      DestUser:   "user",
//                      DestPass:   "pass",
//                }
//                orderId, _ := manager.StartTransfer(transferOrder)
//                manager.WaitOnSessionStop(orderId)
//        }
//
package faspmanager
