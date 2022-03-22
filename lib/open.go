package lib

import (
	"os/exec"
	"runtime"

	"git.sr.ht/~rjarry/aerc/logging"
)

var openBin string = "xdg-open"

func init() {
	if runtime.GOOS == "darwin" {
		openBin = "open"
	}
}

type xdgOpen struct {
	args  []string
	errCh chan (error)
	cmd   *exec.Cmd
}

// NewXDGOpen returns a handler for opening a file via the system handler xdg-open
// or comparable tools on other OSs than Linux
func NewXDGOpen(filename string) *xdgOpen {
	errch := make(chan error, 1)
	return &xdgOpen{
		errCh: errch,
		args:  []string{filename},
	}

}

// SetArgs sets additional arguments to the open command prior to the filename
func (xdg *xdgOpen) SetArgs(args []string) {
	args = append([]string{}, args...) // don't overwrite array of caller
	filename := xdg.args[len(xdg.args)-1]
	xdg.args = append(args, filename)
}

// Start the open handler.
// Returns an error if the command could not be started.
// Use Wait to wait for the commands completion and to check the error.
func (xdg *xdgOpen) Start() error {
	xdg.cmd = exec.Command(openBin, xdg.args...)
	err := xdg.cmd.Start()
	if err != nil {
		xdg.errCh <- err // for callers that just check the error from Wait()
		close(xdg.errCh)
		return err
	}
	go func() {
		defer logging.PanicHandler()

		xdg.errCh <- xdg.cmd.Wait()
		close(xdg.errCh)
	}()
	return nil
}

// Wait for the xdg-open command to complete
// The xdgOpen must have been started
func (xdg *xdgOpen) Wait() error {
	return <-xdg.errCh
}
