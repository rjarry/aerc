package pinentry

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"

	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib/log"
	"git.sr.ht/~rjarry/aerc/lib/ui"
)

var pinentryMode int32 = 0

func Enable() {
	if !config.General.UsePinentry {
		return
	}
	if atomic.SwapInt32(&pinentryMode, 1) == 1 {
		// cannot enter pinentry mode twice
		return
	}
	ui.SuspendScreen()
}

func Disable() {
	if atomic.SwapInt32(&pinentryMode, 0) == 0 {
		// not in pinentry mode
		return
	}
	ui.ResumeScreen()
}

func SetCmdEnv(cmd *exec.Cmd) {
	if cmd == nil || atomic.LoadInt32(&pinentryMode) == 0 {
		return
	}

	env := cmd.Env
	if env == nil {
		env = os.Environ()
	}

	hasTerm := false
	hasGPGTTY := false
	for _, e := range env {
		switch {
		case strings.HasPrefix(strings.ToUpper(e), "TERM="):
			log.Debugf("pinentry: use %v", e)
			hasTerm = true
		case strings.HasPrefix(strings.ToUpper(e), "GPG_TTY="):
			log.Debugf("pinentry: use %v", e)
			hasGPGTTY = true
		}
	}

	if !hasTerm {
		env = append(env, "TERM=xterm-256color")
		log.Debugf("pinentry: set TERM=xterm-256color")
	}

	if !hasGPGTTY {
		tty := ttyname()
		env = append(env, fmt.Sprintf("GPG_TTY=%s", tty))
		log.Debugf("pinentry: set GPG_TTY=%s", tty)
	}

	cmd.Env = env
}
