package faspmanager

import (
	"errors"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
)

func writePortFile(portFilePath string, managementPort int) error {
	basePath := filepath.Dir(portFilePath)
	err := os.MkdirAll(basePath, 0666) // RW permission
	if err != nil {
		return err
	}
	f, err := os.Create(portFilePath)
	if err != nil {
		return err
	}
	defer f.Close()
	fmLog.Println("env_helpers: Writing port file at location: " + portFilePath)
	mgmtPortStr := strconv.Itoa(managementPort)
	_, err = f.Write([]byte(mgmtPortStr))
	return err
}

func getPortFilePath(ascpPath string) (string, error) {
	currUser, err := user.Current()
	if err != nil {
		return "", err
	}
	portFileName := filepath.Base(os.Args[0]) + "." + currUser.Name + "." + strconv.Itoa(os.Getpid()) + ".optport"

	var portFilePath string
	if os := runtime.GOOS; os == "darwin" {
		portFilePath = filepath.Join("/", "Library", "Aspera", "var", "run", "aspera", portFileName)
	} else if os == "windows" {
		portFilePath = filepath.Join(ascpPath, "..", "..", "var", "run", "aspera", portFileName)
	} else if os == "linux" {
		portFilePath = filepath.Join("/", "opt", "aspera", "var", "run", "aspera", portFileName)
	} else {
		// else os is one of: "dragonfly", "freebsd", "netbsd", "openbsd", "plan9", "solaris"
		// See https://golang.org/doc/install/source#environment
		return "", errors.New("Cannot create management port file. Unsupported platform: " + os + ".")
	}
	return portFilePath, nil
}
