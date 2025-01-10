package compose

import (
	"fmt"
	"path/filepath"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/lib/log"
	"github.com/pkg/errors"
)

type Detach struct {
	Path string `opt:"path" required:"false" complete:"CompletePath" desc:"Attachment file path."`
}

func init() {
	commands.Register(Detach{})
}

func (Detach) Description() string {
	return "Detach the file with the given path from the composed email."
}

func (Detach) Context() commands.CommandContext {
	return commands.COMPOSE_EDIT | commands.COMPOSE_REVIEW
}

func (Detach) Aliases() []string {
	return []string{"detach"}
}

func (*Detach) CompletePath(arg string) []string {
	composer, _ := app.SelectedTabContent().(*app.Composer)
	return commands.FilterList(composer.GetAttachments(), arg, nil)
}

func (d Detach) Execute(args []string) error {
	composer, _ := app.SelectedTabContent().(*app.Composer)

	if d.Path == "" {
		// if no attachment is specified, delete the first in the list
		atts := composer.GetAttachments()
		if len(atts) > 0 {
			d.Path = atts[0]
		} else {
			return fmt.Errorf("No attachments to delete")
		}
	}

	return d.removePath(d.Path)
}

func (d Detach) removePath(path string) error {
	composer, _ := app.SelectedTabContent().(*app.Composer)

	// If we don't get an error here, the path was not a pattern.
	if err := composer.DeleteAttachment(path); err == nil {
		log.Debugf("detaching '%s'", path)
		app.PushSuccess(fmt.Sprintf("Detached %s", path))

		return nil
	}

	currentAttachments := composer.GetAttachments()
	detached := make([]string, 0, len(currentAttachments))
	for _, a := range currentAttachments {
		// Don't use filepath.Glob like :attach does. Not all files
		// that match the glob are already attached to the message.
		matches, err := filepath.Match(path, a)
		if err != nil && errors.Is(err, filepath.ErrBadPattern) {
			log.Warnf("failed to parse as globbing pattern: %v", err)
			return err
		}

		if matches {
			log.Debugf("detaching '%s'", a)
			if err := composer.DeleteAttachment(a); err != nil {
				return err
			}

			detached = append(detached, a)
		}
	}

	if len(detached) == 1 {
		app.PushSuccess(fmt.Sprintf("Detached %s", detached[0]))
	} else {
		app.PushSuccess(fmt.Sprintf("Detached %d files", len(detached)))
	}

	return nil
}
