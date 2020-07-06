package lib

import (
	"os/exec"
)

func OpenFile(filename string, onErr func(error)) {
	cmd := exec.Command("open", filename)
	err := cmd.Start()
	if err != nil && onErr != nil {
		onErr(err)
		return
	}

	go func() {
		err := cmd.Wait()
		if err != nil && onErr != nil {
			onErr(err)
		}
	}()
}
