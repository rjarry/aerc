package config

import (
	"errors"
	"os"
	"path/filepath"

	"git.sr.ht/~rjarry/aerc/lib/log"
)

type reloadStore struct {
	binds string
	conf  string
}

var rlst reloadStore

func SetBindsFilename(fn string) {
	log.Debugf("reloader: set binds file: %s", fn)
	rlst.binds = fn
}

func SetConfFilename(fn string) {
	log.Debugf("reloader: set conf file: %s", fn)
	rlst.conf = fn
}

func ReloadBinds() (string, error) {
	f := rlst.binds
	if !exists(f) {
		return f, os.ErrNotExist
	}
	log.Debugf("reload binds file: %s", f)
	return f, parseBindsFromFile(filepath.Dir(f), f)
}

func ReloadConf() (string, error) {
	f := rlst.conf
	if !exists(f) {
		return f, os.ErrNotExist
	}
	log.Debugf("reload conf file: %s", f)

	return f, parseConf(f)
}

func ReloadAccounts() error {
	return errors.New("not implemented")
}

func exists(fn string) bool {
	if _, err := os.Stat(fn); errors.Is(err, os.ErrNotExist) {
		return false
	}
	return true
}
