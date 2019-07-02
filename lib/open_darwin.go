package lib

import (
	"os/exec"
)

func OpenFile(filename string) error {
	cmd := exec.Command("open", filename)
	return cmd.Run()
}
