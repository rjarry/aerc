package lib

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"git.sr.ht/~rjarry/aerc/logging"
)

func XDGOpen(uri string) error {
	return XDGOpenMime(uri, "", nil, nil)
}

func XDGOpenMime(
	uri string, mimeType string,
	openers map[string][]string, args []string,
) error {
	if len(args) == 0 {
		// no explicit command provided, lookup opener from mime type
		opener, ok := openers[mimeType]
		if ok {
			args = opener
		} else {
			// no opener defined in config, fallback to default
			if runtime.GOOS == "darwin" {
				args = append(args, "open")
			} else {
				args = append(args, "xdg-open")
			}
		}
	}

	i := 0
	for ; i < len(args); i++ {
		if strings.Contains(args[i], "{}") {
			break
		}
	}
	if i < len(args) {
		// found {} placeholder in args, replace with uri
		args[i] = strings.Replace(args[i], "{}", uri, 1)
	} else {
		// no {} placeholder in args, add uri at the end
		args = append(args, uri)
	}

	logging.Infof("running command: %v", args)
	cmd := exec.Command(args[0], args[1:]...)
	out, err := cmd.CombinedOutput()
	logging.Debugf("command: %v exited. err=%v out=%s", args, err, out)
	if err != nil {
		return fmt.Errorf("%v: %w", args, err)
	}
	return nil
}
