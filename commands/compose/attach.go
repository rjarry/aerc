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

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib"
	"git.sr.ht/~rjarry/aerc/lib/log"
	"git.sr.ht/~rjarry/aerc/lib/ui"
	"git.sr.ht/~rjarry/aerc/lib/xdg"
	"github.com/pkg/errors"
)

type Attach struct {
	Menu bool   `opt:"-m" desc:"Select files from file-picker-cmd."`
	Name string `opt:"-r" desc:"<name> <cmd...>: Generate attachment from command output."`
	Path string `opt:"path" required:"false" complete:"CompletePath" desc:"Attachment file path."`
	Args string `opt:"..." required:"false"`
}

func init() {
	commands.Register(Attach{})
}

func (Attach) Description() string {
	return "Attach the file at the given path to the email."
}

func (Attach) Context() commands.CommandContext {
	return commands.COMPOSE_EDIT | commands.COMPOSE_REVIEW
}

func (Attach) Aliases() []string {
	return []string{"attach"}
}

func (*Attach) CompletePath(arg string) []string {
	return commands.CompletePath(arg, false)
}

func (a Attach) Execute(args []string) error {
	if a.Menu && a.Name != "" {
		return errors.New("-m and -r are mutually exclusive")
	}
	switch {
	case a.Menu:
		return a.openMenu()
	case a.Name != "":
		if a.Path == "" {
			return errors.New("command is required")
		}
		return a.readCommand()
	default:
		if a.Args != "" {
			return errors.New("only a single path is supported")
		}
		return a.addPath(a.Path)
	}
}

func (a Attach) addPath(path string) error {
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

	composer, _ := app.SelectedTabContent().(*app.Composer)
	for _, attach := range attachments {
		log.Debugf("attaching '%s'", attach)

		pathinfo, err := os.Stat(attach)
		if err != nil {
			log.Errorf("failed to stat file: %v", err)
			app.PushError(err.Error())
			return err
		} else if pathinfo.IsDir() && len(attachments) == 1 {
			app.PushError("Attachment must be a file, not a directory")
			return nil
		}

		composer.AddAttachment(attach)
	}

	if len(attachments) == 1 {
		app.PushSuccess(fmt.Sprintf("Attached %s", path))
	} else {
		app.PushSuccess(fmt.Sprintf("Attached %d files", len(attachments)))
	}

	return nil
}

func (a Attach) openMenu() error {
	filePickerCmd := config.Compose.FilePickerCmd
	if filePickerCmd == "" {
		return fmt.Errorf("no file-picker-cmd defined")
	}

	if strings.Contains(filePickerCmd, "%s") {
		filePickerCmd = strings.ReplaceAll(filePickerCmd, "%s", a.Path)
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

	t, err := app.NewTerminal(filepicker)
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

		app.CloseDialog()

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
			err := a.addPath(f)
			if err != nil {
				log.Errorf("attach failed for file %s: %v", f, err)
			}

		}
	}

	app.AddDialog(app.DefaultDialog(
		ui.NewBox(t, "File Picker", "", app.SelectedAccountUiConfig()),
	))

	return nil
}

func (a Attach) readCommand() error {
	cmd := exec.Command("sh", "-c", a.Path+" "+a.Args)

	data, err := cmd.Output()
	if err != nil {
		return errors.Wrap(err, "Output")
	}

	reader := bufio.NewReader(bytes.NewReader(data))

	mimeType, mimeParams, err := lib.FindMimeType(a.Name, reader)
	if err != nil {
		return errors.Wrap(err, "FindMimeType")
	}

	mimeParams["name"] = a.Name

	composer, _ := app.SelectedTabContent().(*app.Composer)
	err = composer.AddPartAttachment(a.Name, mimeType, mimeParams, reader)
	if err != nil {
		return errors.Wrap(err, "AddPartAttachment")
	}

	app.PushSuccess(fmt.Sprintf("Attached %s", a.Name))

	return nil
}
