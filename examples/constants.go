package examples

import (
	"errors"
	"path/filepath"
	"runtime"
)

var (
	ASCP			= ""
	ASCP4			= ""
	SMALL_LOCAL_FILE	= ""
	MEDIUM_LOCAL_FILE	= ""
	REMOTE_DIR		= "/"
	REMOTE_HOST		= "169.45.166.110"
	REMOTE_USER		= "comcast"
	REMOTE_PASS		= "Clever2safe"
)

func init() {
	
	var platform string
	if os := runtime.GOOS; os == "darwin" {
		platform = "macosx-64"
	} else if os == "linux" {
		platform = "linux-64"
	} else {
		panic(errors.New("No ascp installs included for platform " + os + "."))
	}

	var err error

	ASCP, err = filepath.Abs(filepath.Join("..", "..", "bin", platform, "ascp"))
	if err != nil {
		panic(err)
	}
	
	ASCP4, err = filepath.Abs(filepath.Join("..", "..", "bin", platform, "ascp4"))
	if err != nil {
		panic(err)
	}

	SMALL_LOCAL_FILE, err = filepath.Abs(filepath.Join("..", "etc", "200KB_FILE"))
	if err != nil {
		panic(err)
	}
	
	MEDIUM_LOCAL_FILE, err = filepath.Abs(filepath.Join("..", "etc", "10MB_FILE"))
	if err != nil {
		panic(err)
	}	

}
