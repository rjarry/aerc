package xdg

import (
	"os"
	"os/user"
	"path"
	"strings"

	"git.sr.ht/~rjarry/aerc/log"
)

// assign to a local var to allow mocking in unit tests
var currentUser = user.Current

// Get the current user home directory (first from the $HOME env var and
// fallback on calling getpwuid_r() from libc if $HOME is unset).
func HomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		u, e := currentUser()
		if e == nil {
			home = u.HomeDir
		} else {
			log.Errorf("HomeDir: %s (while handling %s)", e, err)
		}
	}
	return home
}

// Replace ~ with the current user's home dir
func ExpandHome(fragments ...string) string {
	home := HomeDir()
	res := path.Join(fragments...)
	if strings.HasPrefix(res, "~/") || res == "~" {
		res = home + strings.TrimPrefix(res, "~")
	}
	return res
}

// Replace $HOME with ~ (inverse function of ExpandHome)
func TildeHome(path string) string {
	home := HomeDir()
	if strings.HasPrefix(path, home+"/") || path == home {
		path = "~" + strings.TrimPrefix(path, home)
	}
	return path
}
