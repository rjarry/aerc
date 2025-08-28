package hooks

import (
	"bytes"
	"os"
	"os/exec"

	"git.sr.ht/~rjarry/aerc/lib/log"
)

func RunHook(h HookType) error {
	cmd := h.Cmd()
	if cmd == "" {
		return nil
	}
	env := h.Env()
	log.Debugf("hooks: running %T command %q (env %v)", h, cmd, env)

	proc := exec.Command("sh", "-c", cmd)
	var outb, errb bytes.Buffer
	proc.Stdout = &outb
	proc.Stderr = &errb
	proc.Env = os.Environ()
	proc.Env = append(proc.Env, env...)
	err := proc.Run()
	log.Tracef("hooks: %q stdout: %s", cmd, outb.String())
	if err != nil {
		log.Errorf("hooks:%q stderr: %s", cmd, errb.String())
	}
	return err
}
