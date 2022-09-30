package lib

import (
	"fmt"
	"os/exec"
	"runtime"

	"git.sr.ht/~rjarry/aerc/logging"
)

func XDGOpen(uri string) error {
	openBin := "xdg-open"
	if runtime.GOOS == "darwin" {
		openBin = "open"
	}
	args := []string{openBin, uri}
	logging.Infof("running command: %v", args)
	cmd := exec.Command(args[0], args[1:]...)
	out, err := cmd.CombinedOutput()
	logging.Debugf("command: %v exited. err=%v out=%s", args, err, out)
	if err != nil {
		return fmt.Errorf("%v: %w", args, err)
	}
	return nil
}
