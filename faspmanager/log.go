package faspmanager

import (
	"io/ioutil"
	"log"
)

var (
	fmLog   *log.Logger = log.New(ioutil.Discard, "", 0) // Logs information from FASP Manager.
	mgmtLog *log.Logger = log.New(ioutil.Discard, "", 0) // Logs ascp management messages.
)
