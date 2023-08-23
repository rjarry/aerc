package compose

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib"
	"git.sr.ht/~rjarry/aerc/lib/ui"
	"git.sr.ht/~rjarry/aerc/lib/xdg"
	"git.sr.ht/~rjarry/aerc/log"
	"git.sr.ht/~rjarry/aerc/widgets"
	"github.com/pkg/errors"

	"git.sr.ht/~sircmpwn/getopt"
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
	var (
		menu bool
		read bool
	)

	opts, optind, err := getopt.Getopts(args, "mr")
	if err != nil {
		return err
	}

	for _, opt := range opts {
		switch opt.Option {
		case 'm':
			if read {
				return errors.New("-m and -r are mutually exclusive")
			}
			menu = true
		case 'r':
			if menu {
				return errors.New("-m and -r are mutually exclusive")
			}
			read = true
		}
	}

	args = args[optind:]

	if menu {
		return a.openMenu(aerc, args)
	}

	if read {
		if len(args) < 2 {
			return fmt.Errorf("Usage: :attach -r <name> <cmd> [args...]")
		}
		return a.readCommand(aerc, args[0], args[1:])
	}

	if len(args) == 0 {
		return fmt.Errorf("Usage: :attach <path>")
	}
	return a.addPath(aerc, strings.Join(args, " "))
}

func (a Attach) addPath(aerc *widgets.Aerc, path string) error {
	path = xdg.ExpandHome(path)
	attachments, err := filepath.Glob(path)
	if err != nil && errors.Is(err, filepath.ErrBadPattern) {
		log.Warnf("failed to parse as globbing pattern: %v", err)
		attachments = []string{path}
	}

	if !strings.HasPrefix(path, ".") && !strings.Contains(path, "/.") {
		log.Debugf("removing hidden files from glob results")
		for i := len(attachments) - 1; i >= 0; i-- {
			if strings.HasPrefix(filepath.Base(attachments[i]), ".") {
				if i == len(attachments)-1 {
					attachments = attachments[:i]
					continue
				}
				attachments = append(attachments[:i], attachments[i+1:]...)
			}
		}
	}

	composer, _ := aerc.SelectedTabContent().(*widgets.Composer)
	for _, attach := range attachments {
		log.Debugf("attaching '%s'", attach)

		pathinfo, err := os.Stat(attach)
		if err != nil {
			log.Errorf("failed to stat file: %v", err)
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
	filePickerCmd := config.Compose.FilePickerCmd
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

	var filepicker *exec.Cmd
	if strings.Contains(filePickerCmd, "%f") {
		filePickerCmd = strings.ReplaceAll(filePickerCmd, "%f", picks.Name())
		filepicker = exec.Command("sh", "-c", filePickerCmd)
	} else {
		filepicker = exec.Command("sh", "-c", filePickerCmd+" >&3")
		filepicker.ExtraFiles = append(filepicker.ExtraFiles, picks)
	}

	t, err := widgets.NewTerminal(filepicker)
	if err != nil {
		return err
	}
	t.Focus(true)
	t.OnClose = func(err error) {
		defer func() {
			if err := picks.Close(); err != nil {
				log.Errorf("error closing file: %v", err)
			}
			if err := os.Remove(picks.Name()); err != nil {
				log.Errorf("could not remove tmp file: %v", err)
			}
		}()

		aerc.CloseDialog()

		if err != nil {
			log.Errorf("terminal closed with error: %v", err)
			return
		}

		_, err = picks.Seek(0, io.SeekStart)
		if err != nil {
			log.Errorf("seek failed: %v", err)
			return
		}

		scanner := bufio.NewScanner(picks)
		for scanner.Scan() {
			f := strings.TrimSpace(scanner.Text())
			if _, err := os.Stat(f); err != nil {
				continue
			}
			log.Tracef("File picker attaches: %v", f)
			err := a.addPath(aerc, f)
			if err != nil {
				log.Errorf("attach failed for file %s: %v", f, err)
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

func (a Attach) readCommand(aerc *widgets.Aerc, name string, args []string) error {
	args = append([]string{"-c"}, args...)
	cmd := exec.Command("sh", args...)

	data, err := cmd.Output()
	if err != nil {
		return errors.Wrap(err, "Output")
	}

	reader := bufio.NewReader(bytes.NewReader(data))

	mimeType, mimeParams, err := lib.FindMimeType(name, reader)
	if err != nil {
		return errors.Wrap(err, "FindMimeType")
	}

	mimeParams["name"] = name

	composer, _ := aerc.SelectedTabContent().(*widgets.Composer)
	err = composer.AddPartAttachment(name, mimeType, mimeParams, reader)
	if err != nil {
		return errors.Wrap(err, "AddPartAttachment")
	}

	aerc.PushSuccess(fmt.Sprintf("Attached %s", name))

	return nil
}
