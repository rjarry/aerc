package hooks

import (
	"os"
	"os/exec"

	"git.sr.ht/~rjarry/aerc/log"
)

func RunHook(h HookType) error {
	cmd := h.Cmd()
	if cmd == "" {
		return nil
	}
	env := h.Env()
	log.Debugf("hooks: running command %q (env %v)", cmd, env)

	proc := exec.Command("sh", "-c", cmd)
	proc.Env = os.Environ()
	proc.Env = append(proc.Env, env...)
	return proc.Run()
}
