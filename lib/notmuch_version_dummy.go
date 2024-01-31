//go:build !notmuch
// +build !notmuch

package lib

func NotmuchVersion() (string, bool) {
	return "", false
}
