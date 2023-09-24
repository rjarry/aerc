package lib

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"git.sr.ht/~rjarry/go-opt"
	"github.com/danwakefield/fnmatch"

	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/log"
)

func XDGOpenMime(
	uri string, mimeType string, args string,
) error {
	if len(args) == 0 {
		// no explicit command provided, lookup opener from mime type
		for _, o := range config.Openers {
			if fnmatch.Match(o.Mime, mimeType, 0) {
				args = o.Args
				break
			}
		}
	}
	if len(args) == 0 {
		// no opener defined in config, fallback to default
		if runtime.GOOS == "darwin" {
			args = "open"
		} else {
			args = "xdg-open"
		}
	}

	// Escape URI special characters
	uri = opt.QuoteArg(uri)
	if strings.Contains(args, "{}") {
		// found {} placeholder in args, replace with uri
		args = strings.Replace(args, "{}", uri, 1)
	} else {
		// no {} placeholder in args, add uri at the end
		args = args + " " + uri
	}

	log.Tracef("running command: %v", args)
	cmd := exec.Command("sh", "-c", args)
	out, err := cmd.CombinedOutput()
	log.Debugf("command: %v exited. err=%v out=%s", args, err, out)
	if err != nil {
		return fmt.Errorf("%v: %w", args, err)
	}
	return nil
}
