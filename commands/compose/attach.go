package compose

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/lib/ui"
	"git.sr.ht/~rjarry/aerc/logging"
	"git.sr.ht/~rjarry/aerc/widgets"
	"github.com/mitchellh/go-homedir"
)

type Attach struct{}

func init() {
	register(Attach{})
}

func (Attach) Aliases() []string {
	return []string{"attach"}
}

func (Attach) Complete(aerc *widgets.Aerc, args []string) []string {
	path := strings.Join(args, " ")
	return commands.CompletePath(path)
}

func (a Attach) Execute(aerc *widgets.Aerc, args []string) error {
	if len(args) == 1 {
		return fmt.Errorf("Usage: :attach <path>")
	}

	if args[1] == "-m" {
		return a.openMenu(aerc, args[2:])
	}

	return a.addPath(aerc, strings.Join(args[1:], " "))
}

func (a Attach) addPath(aerc *widgets.Aerc, path string) error {
	path, err := homedir.Expand(path)
	if err != nil {
		logging.Errorf("failed to expand path '%s': %v", path, err)
		aerc.PushError(err.Error())
		return err
	}

	logging.Debugf("attaching %s", path)

	attachments, err := filepath.Glob(path)
	if err != nil && errors.Is(err, filepath.ErrBadPattern) {
		logging.Warnf("failed to parse as globbing pattern: %v", err)
		attachments = []string{path}
	}

	logging.Debugf("filenames: %v", attachments)

	composer, _ := aerc.SelectedTabContent().(*widgets.Composer)
	for _, attach := range attachments {
		logging.Debugf("attaching '%s'", attach)

		pathinfo, err := os.Stat(attach)
		if err != nil {
			logging.Errorf("failed to stat file: %v", err)
			aerc.PushError(err.Error())
			return err
		} else if pathinfo.IsDir() && len(attachments) == 1 {
			aerc.PushError("Attachment must be a file, not a directory")
			return nil
		}

		composer.AddAttachment(attach)
	}

	if len(attachments) == 1 {
		aerc.PushSuccess(fmt.Sprintf("Attached %s", path))
	} else {
		aerc.PushSuccess(fmt.Sprintf("Attached %d files", len(attachments)))
	}

	return nil
}

func (a Attach) openMenu(aerc *widgets.Aerc, args []string) error {
	filePickerCmd := aerc.Config().Compose.FilePickerCmd
	if filePickerCmd == "" {
		return fmt.Errorf("no file-picker-cmd defined")
	}

	if strings.Contains(filePickerCmd, "%s") {
		verb := ""
		if len(args) > 0 {
			verb = args[0]
		}
		filePickerCmd = strings.ReplaceAll(filePickerCmd, "%s", verb)
	}

	picks, err := os.CreateTemp("", "aerc-filepicker-*")
	if err != nil {
		return err
	}

	filepicker := exec.Command("sh", "-c", filePickerCmd+" >&3")
	filepicker.ExtraFiles = append(filepicker.ExtraFiles, picks)

	t, err := widgets.NewTerminal(filepicker)
	if err != nil {
		return err
	}
	t.OnClose = func(err error) {
		defer func() {
			if err := picks.Close(); err != nil {
				logging.Errorf("error closing file: %v", err)
			}
			if err := os.Remove(picks.Name()); err != nil {
				logging.Errorf("could not remove tmp file: %v", err)
			}
		}()

		aerc.CloseDialog()

		if err != nil {
			logging.Errorf("terminal closed with error: %v", err)
			return
		}

		_, err = picks.Seek(0, io.SeekStart)
		if err != nil {
			logging.Errorf("seek failed: %v", err)
		}

		scanner := bufio.NewScanner(picks)
		for scanner.Scan() {
			f := strings.TrimSpace(scanner.Text())
			if _, err := os.Stat(f); err != nil {
				continue
			}
			logging.Infof("File picker attaches: %v", f)
			err := a.addPath(aerc, f)
			if err != nil {
				logging.Errorf("attach failed "+
					"for file %s: %v", f, err)
			}

		}
	}

	aerc.AddDialog(widgets.NewDialog(
		ui.NewBox(t, "File Picker", "", aerc.SelectedAccountUiConfig()),
		// start pos on screen
		func(h int) int {
			return h / 8
		},
		// dialog height
		func(h int) int {
			return h - 2*h/8
		},
	))

	return nil
}
