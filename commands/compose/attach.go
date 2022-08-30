package compose

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"git.sr.ht/~rjarry/aerc/commands"
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

func (Attach) Execute(aerc *widgets.Aerc, args []string) error {
	if len(args) == 1 {
		return fmt.Errorf("Usage: :attach <path>")
	}

	path := strings.Join(args[1:], " ")
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
