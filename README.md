# Aspera FASP Manager Go SDK

FASP Manager offers programmatic initiation, monitoring, and control of FASP
based transfers, including stats collection, dynamic rate control, and
stream-to-file capabilities.

## PACKAGE CONTENTS

* faspmanager-sdk-go/faspmanager: Implementation of the FASP Manager API
* faspmanager-sdk-go/examples:    Example usage
* faspmanager-sdk-go/bin:         Required Aspera ascp binaries and license files

## GETTING STARTED

The easiest way to get familiar with the FASP Manager API is to run a sample
program.  The following steps will guide you through starting a transfer with an
Aspera demo server.

1. Install the Go programming language and set up a Go workspace directory.  Set
   the GOPATH environment variable to point to this directory.
   
2. Clone faspmanager-sdk-go into subdirectory
   src/github.com/pacgenesis/faspmanager-sdk-go within the Go workspace
   directory. Required Aspera ascp binaries and configuration files for macosx
   and linux 64 bit platforms are included.

3. From the command line, navigate into this directory and run 'go get'. Ensure
   dependencies install.
   
4. Navigate into any of the subdirectories under faspmanager-sdk-go/examples and
   run 'go run main.go'. You should see output to the terminal periodically
   showing the status of a transfer as well as a notification upon its
   successful completion.

## DOCUMENTATION

Documentation can be generated and accessed using the godoc command line tool.

From the command line, navigate into the faspmanager-sdk-go/faspmanager
directory and start a godoc server ("godoc -http=:6060"). Then use a web browser
to open the documentation site ("localhost/6060") and click the "packages"
button. The faspmanager-sdk-go library will be listed along the left.

Alternatively, generate HTML documentation by navigating into the
faspmanager-sdk-go/faspmanager directory and typing "godoc -html . >
doc.html". This generates a doc.html file.

## VERSIONS

This FASP Manager SDK requires ascp (or ascp.exe) version 3.7 or later.
However, certain features, such as the stream-to-file capabilities, may require
a newer version.

## LOGGING

The SDK uses instances of log.Logger initialized in log.go. By default, log
output is discarded, but this can be changed by setting the loggers' output
appropriately.
