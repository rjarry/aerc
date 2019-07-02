// +build !darwin

package lib

import (
	"os/exec"
)

func OpenFile(filename string) error {
	cmd := exec.Command("xdg-open", filename)
	return cmd.Run()
}
