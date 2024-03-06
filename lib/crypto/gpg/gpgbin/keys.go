package gpgbin

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// GetPrivateKeyId runs gpg --list-secret-keys s
func GetPrivateKeyId(s string) (string, error) {
	private := true
	id := getKeyId(s, private)
	if id == "" {
		return "", fmt.Errorf("no private key found")
	}
	return id, nil
}

// GetKeyId runs gpg --list-keys s
func GetKeyId(s string) (string, error) {
	private := false
	id := getKeyId(s, private)
	if id == "" {
		return "", fmt.Errorf("no public key found")
	}
	return id, nil
}

// ExportPublicKey exports the public key identified by k in armor format
func ExportPublicKey(k string) (io.Reader, error) {
	cmd := exec.Command("gpg", "--armor",
		"--export-options", "export-minimal", "--export", k)

	var outbuf bytes.Buffer
	var stderr strings.Builder
	cmd.Stdout = &outbuf
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("gpg: export failed: %w", err)
	}
	if strings.Contains(stderr.String(), "gpg") {
		return nil, fmt.Errorf("gpg: error exporting key")
	}
	return &outbuf, nil
}
