package faspmanager

import (
	"strconv"
	"strings"
	"testing"
)

func TestBuildAscpCommandLine_NilTransferOptions(t *testing.T) {
	ascpOpts := &AscpOptions{
		SourcePaths: []string{"source.txt"},
		DestPaths:   []string{"dest.txt"},
		RemoteHost:  "localhost",
		RemoteUser:  "user",
		RemotePass:  "pass",
		Direction:   UploadDirection,
	}

	_, err := ascpOpts.buildAscpCommandLine()
	if err != nil {
		t.Fatal("buildAscpCommandLine should allow nil TransferOpts.")
	}
}

func TestBuildAscpCommandLine_InvalidDatagramSize(t *testing.T) {
	ascpOpts := &AscpOptions{
		TransferOpts: &TransferOptions{
			DatagramSize: -1,
		},
	}

	_, err := ascpOpts.buildAscpCommandLine()
	if err == nil {
		t.Fail()
	}
}

func TestBuildAscpCommandLine_InvalidRates(t *testing.T) {
	ascpOpts := &AscpOptions{
		TransferOpts: &TransferOptions{
			MinRateKbps:    5000,
			TargetRateKbps: 3000,
		},
	}

	_, err := ascpOpts.buildAscpCommandLine()
	if err == nil {
		t.Fail()
	}

	ascpOpts = &AscpOptions{
		TransferOpts: &TransferOptions{
			AutoDetectCapacity: true,
			MinRateKbps:        1000,
			TargetRateKbps:     3000,
		},
	}

	_, err = ascpOpts.buildAscpCommandLine()
	if err == nil {
		t.Fail()
	}
}

func TestBuildAscpCommandLine_InvalidPorts(t *testing.T) {
	ascpOpts := &AscpOptions{
		TransferOpts: &TransferOptions{
			TcpPort: -1,
		},
	}

	_, err := ascpOpts.buildAscpCommandLine()
	if err == nil {
		t.Fail()
	}

	ascpOpts = &AscpOptions{
		TransferOpts: &TransferOptions{
			UdpPort: -1,
		},
	}

	_, err = ascpOpts.buildAscpCommandLine()
	if err == nil {
		t.Fail()
	}
}

func TestBuildAscpCommandLine_InvalidRetryTimeout(t *testing.T) {
	ascpOpts := &AscpOptions{
		TransferOpts: &TransferOptions{
			RetryTimeoutS: -1,
		},
	}

	_, err := ascpOpts.buildAscpCommandLine()
	if err == nil {
		t.Fail()
	}
}

func TestBuildAscpCommandLine_InvalidTransferRetry(t *testing.T) {
	ascpOpts := &AscpOptions{
		TransferOpts: &TransferOptions{
			TransferRetry: -1,
		},
	}

	_, err := ascpOpts.buildAscpCommandLine()
	if err == nil {
		t.Fail()
	}
}

func TestBuildAscpCommandLine_InvalidHttpFallback(t *testing.T) {
	ascpOpts := &AscpOptions{
		TransferOpts: &TransferOptions{
			HttpFallback: &HttpFallback{
				HttpsKeyFileName:  "__nonexistent",
				HttpsCertFileName: "__nonexistent",
			},
		},
	}

	_, err := ascpOpts.buildAscpCommandLine()
	if err == nil {
		t.Fail()
	}
}

func TestBuildAscpCommandLine_InvalidMoveAfterTransfer(t *testing.T) {
	ascpOpts := &AscpOptions{
		TransferOpts: &TransferOptions{
			MoveAfterTransferPath:   "somepath",
			RemoveFileAfterTransfer: true,
		},
	}

	_, err := ascpOpts.buildAscpCommandLine()
	if err == nil {
		t.Fail()
	}
}

func TestBuildAscpCommandLine_Cipher(t *testing.T) {
	testCipherType := func(c CipherType, cstr string) {
		ascpOpts := &AscpOptions{
			TransferOpts: &TransferOptions{
				Cipher: c,
			},
		}

		cmd, err := ascpOpts.buildAscpCommandLine()
		if err != nil {
			t.Fail()
		}

		cmdMap := map[string]bool{}
		for _, option := range cmd {
			cmdMap[option] = true
		}

		if !cmdMap[cstr] {
			t.Fatalf("Missing cipher string. Expected: %s", cstr)
		}
	}
	testCipherType(AES_128_CIPHER, "-caes128")
	testCipherType(AES_192_CIPHER, "-caes192")
	testCipherType(AES_256_CIPHER, "-caes256")
}

func TestBuildAscpCommandLine_FileChecksum(t *testing.T) {
	testChecksumType := func(c FileChecksumType, cstr string) {
		ascpOpts := &AscpOptions{
			TransferOpts: &TransferOptions{
				FileChecksum: c,
			},
		}

		cmd, err := ascpOpts.buildAscpCommandLine()
		if err != nil {
			t.Fail()
		}

		cmdMap := map[string]bool{}
		for _, option := range cmd {
			cmdMap[option] = true
		}

		if !cmdMap[cstr] {
			t.Fatalf("Missing file checksum string. Expected: %s", cstr)
		}
	}
	testChecksumType(SHA512_CHECKSUM, "--file-checksum=sha-512")
	testChecksumType(SHA384_CHECKSUM, "--file-checksum=sha-384")
	testChecksumType(SHA256_CHECKSUM, "--file-checksum=sha-256")
	testChecksumType(SHA1_CHECKSUM, "--file-checksum=sha1")
	testChecksumType(MD5_CHECKSUM, "--file-checksum=md5")
}

func TestBuildAscpCommandLine_DefaultTransferOpts(t *testing.T) {
	ascpOpts := &AscpOptions{
		SourcePaths:  nil,
		DestPaths:    nil,
		RemoteHost:   "127.0.0.1",
		RemoteUser:   "user",
		RemotePass:   "pass",
		Persistent:   true,
		Direction:    UploadDirection,
		TransferOpts: &TransferOptions{},
	}

	cmd, err := ascpOpts.buildAscpCommandLine()
	if err != nil {
		t.Fatal(err)
	}

	expected := []string{
		"--keepalive",
		"--mode=send",
		"--host=127.0.0.1",
		"--user=user", "-q",
		"--policy=fair",
		"-O " + strconv.Itoa(DefaultUdpPort),
		"-P " + strconv.Itoa(DefaultTcpPort),
		"--retry-timeout=0",
		"-l" + strconv.Itoa(DefaultTargetRate),
		"--tags64",
		"--overwrite=diff",
	}

	cmdMap := map[string]bool{}
	for _, option := range cmd {
		cmdMap[option] = true
	}

	for _, option := range expected {
		if !cmdMap[option] {
			t.Fatalf("Missing option: %v. Command: %v", option, cmd)
		}
	}
}

func TestBuildAscpCommandLine_AllTransferOpts(t *testing.T) {
	ascpOpts := &AscpOptions{
		SourcePaths: nil,
		DestPaths:   nil,
		RemoteHost:  "127.0.0.1",
		RemoteUser:  "user",
		RemotePass:  "pass",
		Persistent:  true,
		Direction:   DownloadDirection,
		TransferOpts: &TransferOptions{
			AutoDetectCapacity:           false,
			Cookie:                       "cookie",
			CreateDirs:                   true,
			DatagramSize:                 1400,
			DisableEncryption:            true,
			LocalLogDir:                  "locallog",
			MinRateKbps:                  3000,
			Policy:                       HIGH_TRANSFER_POLICY,
			PreserveDates:                true,
			RemoteLogDir:                 "remotelog",
			RetryTimeoutS:                2,
			SymLinkPolicy:                COPY_SYM_LINK_POLICY,
			ResumeCheck:                  FULL_CHECKSUM_RESUME,
			SourceBase:                   "base",
			TargetRateKbps:               20000,
			TcpPort:                      20,
			Token:                        "token",
			UdpPort:                      33000,
			ContentProtectionPassphrase:  "passphrase",
			PreCalculateJobSize:          true,
			DeleteSource:                 FILES_ONLY_DELETE_SOURCE,
			RemoveFileAfterTransfer:      true,
			SkipSpecialFiles:             true,
			PreserveFileOwnerUid:         true,
			PreserveFileOwnerGid:         true,
			OverwritePolicy:              ALWAYS_OVERWRITE_POLICY,
			ReadSize:                     5000,
			WriteSize:                    5000,
			PrePostCommandPath:           "path",
			AlternateConfigFileName:      "config",
			PartialFileSuffix:            "partial",
			RetransmissionRequestMaxSize: 1400,
			HttpFallback: &HttpFallback{
				EncodeAllAsJpeg: true,
				HttpPort:        23,
				//HttpsKeyFileName: "keyfile",
				//HttpsCertFileName: "certfile",
				HttpProxyAddressHost: "proxyhost",
				HttpProxyAddressPort: 200,
			},
			// Cannot be set in conjunction with RemoveFileAfterTransfer
			//MoveAfterTransferPath: "moveafter",
			DestinationRoot:      "destroot",
			ReportSkippedFiles:   true,
			TransferRetry:        2,
			TransferId:           "transferid",
			PreserveXAttrs:       NONE_PRESERVE_MODE,
			RemotePreserveXAttrs: NONE_PRESERVE_MODE,
			// Cannot be set in conjunction with Http fallback.
			//DeleteBeforeTransfer: true,
			PreserveModificationTime:   true,
			PreserveAccessTime:         true,
			PreserveSourceAccessTime:   true,
			IgnoreHostKey:              true,
			ExcludeNewerThan:           100,
			ExcludeOlderThan:           100,
			SaveBeforeOverwriteEnabled: true,
			PreserveCreationTime:       true,
			ExcludePatterns:            []string{"_excludethis"},
			SkipDirTraversalDupes:      true,
			ApplyLocalDocroot:          true,
			MultiSessionThreshold:      1000,
			RemoveEmptySourceDirectory: true,
			SourcePrefix:               "sourceprefix",
			CheckSshFingerprint:        "fingerprint",
			PreserveAcls:               NATIVE_PRESERVE_MODE,
			RemotePreserveAcls:         METAFILE_PRESERVE_MODE,
			FileManifestPath:           "manifestpath",
			FileManifestFormat:         TEXT_FILE_MANIFEST_FORMAT,
		},
	}

	cmd, err := ascpOpts.buildAscpCommandLine()
	if err != nil {
		t.Fatal(err)
	}

	expected := map[string]bool{
		"--keepalive":           true,
		"--mode=recv":           true,
		"--host=127.0.0.1":      true,
		"--user=user":           true,
		"-q":                    true,
		"--policy=high":         true,
		"-O 33000":              true,
		"-P 20":                 true,
		"--retry-timeout=2":     true,
		"-l20000":               true,
		"-m3000":                true,
		"--tags64":              true,
		"--symbolic-links=copy": true,
		"-k3": true,
		"-p":  true,
		"--skip-special-files":              true,
		"--preserve-file-owner-uid":         true,
		"--preserve-file-owner-gid":         true,
		"-Z1400":                            true,
		"-d":                                true,
		"-Llocallog":                        true,
		"-Rremotelog":                       true,
		"--src-base=base":                   true,
		"-g 5000":                           true,
		"-G 5000":                           true,
		"--remove-after-transfer":           true,
		"--precalculate-job-size":           true,
		"-e path":                           true,
		"-f config":                         true,
		"--partial-file-suffix=partial":     true,
		"-X1400":                            true,
		"-y1":                               true,
		"-T":                                true,
		"-j1":                               true,
		"-t23":                              true,
		"-xproxyhost:200":                   true,
		"--preserve-modification-time":      true,
		"--preserve-access-time":            true,
		"--preserve-source-access-time":     true,
		"--ignore-host-key":                 true,
		"--file-crypt=decrypt":              true,
		"--overwrite=always":                true,
		"--exclude-newer-than=100":          true,
		"--exclude-older-than=100":          true,
		"--save-before-overwrite":           true,
		"--skip-dir-traversal-dupes":        true,
		"--apply-local-docroot":             true,
		"--multi-session-threshold=1000":    true,
		"--remove-empty-source-directory":   true,
		"--source-prefix=sourceprefix":      true,
		"--check-sshfp=fingerprint":         true,
		"--preserve-xattrs=none":            true,
		"--remote-preserve-xattrs=none":     true,
		"--preserve-acls=native":            true,
		"-E_excludethis":                    true,
		"--file-manifest-path=manifestpath": true,
		"--file-manifest=text":              true,
		"--remote-preserve-acls=metafile":   true,
		"--preserve-creation-time":          true,
		"destroot":                          true,
	}

	cmdMap := map[string]bool{}
	for _, option := range cmd {
		cmdMap[option] = true
	}
	for option, _ := range expected {
		if !cmdMap[option] {
			t.Fatalf("Missing option: %v. Command: %v", option, cmd)
		}
	}
}

func TestBuildAscpCommandLine_SingleFile(t *testing.T) {
	ascpOpts := &AscpOptions{
		SourcePaths:  []string{"a"},
		DestPaths:    []string{"b"},
		RemoteHost:   "127.0.0.1",
		RemoteUser:   "user",
		RemotePass:   "pass",
		Direction:    UploadDirection,
		TransferOpts: &TransferOptions{},
	}

	cmd, err := ascpOpts.buildAscpCommandLine()
	if err != nil {
		t.Fatal(err)
	}

	for _, v := range cmd {
		if strings.Index(v, "--file-list=") == 0 {
			return
		}
	}

	t.Fatal("No --file-list option found.")
}

func TestBuildAscpCommandLine_FilePairs(t *testing.T) {
	ascpOpts := &AscpOptions{
		SourcePaths:  []string{"a", "b"},
		DestPaths:    []string{"c", "d"},
		RemoteHost:   "127.0.0.1",
		RemoteUser:   "user",
		RemotePass:   "pass",
		Direction:    UploadDirection,
		TransferOpts: &TransferOptions{},
	}

	cmd, err := ascpOpts.buildAscpCommandLine()
	if err != nil {
		t.Fatal(err)
	}

	for _, v := range cmd {
		if strings.Index(v, "--file-pair-list=") == 0 {
			return
		}
	}

	t.Fatal("No --file-pair-list option found.")
}

func TestBuildAscpCommandLine_DetectsIpv4Addr(t *testing.T) {
	ascpOpts := &AscpOptions{
		RemoteHost:   "192.0.2.1",
		TransferOpts: &TransferOptions{},
	}

	cmd, err := ascpOpts.buildAscpCommandLine()
	if err != nil {
		t.Fatal(err)
	}

	for _, v := range cmd {
		if v == "-6" {
			t.Fatal("-6 option appended with Ipv4 address.")
		}
	}
}

func TestBuildAscpCommandLine_DetectsIpv6Addr(t *testing.T) {
	ascpOpts := &AscpOptions{
		RemoteHost:   "2001:0db8:85a3:0000:0000:8a2e:0370:7334",
		TransferOpts: &TransferOptions{},
	}

	cmd, err := ascpOpts.buildAscpCommandLine()
	if err != nil {
		t.Fatal(err)
	}

	for _, v := range cmd {
		if v == "-6" {
			return
		}
	}

	t.Fatal("-6 option not found.")
}
