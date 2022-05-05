package gpgbin

import "fmt"

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
