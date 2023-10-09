package compose

import (
	"fmt"
	"strings"

	"git.sr.ht/~rjarry/aerc/app"
)

type Detach struct{}

func init() {
	register(Detach{})
}

func (Detach) Aliases() []string {
	return []string{"detach"}
}

func (Detach) Complete(aerc *app.Aerc, args []string) []string {
	composer, _ := aerc.SelectedTabContent().(*app.Composer)
	return composer.GetAttachments()
}

func (Detach) Execute(aerc *app.Aerc, args []string) error {
	var path string
	composer, _ := aerc.SelectedTabContent().(*app.Composer)

	if len(args) > 1 {
		path = strings.Join(args[1:], " ")
	} else {
		// if no attachment is specified, delete the first in the list
		atts := composer.GetAttachments()
		if len(atts) > 0 {
			path = atts[0]
		} else {
			return fmt.Errorf("No attachments to delete")
		}
	}

	if err := composer.DeleteAttachment(path); err != nil {
		return err
	}

	aerc.PushSuccess(fmt.Sprintf("Detached %s", path))

	return nil
}
