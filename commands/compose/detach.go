package compose

import (
	"fmt"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
)

type Detach struct {
	Path string `opt:"path" required:"false" complete:"CompletePath"`
}

func init() {
	register(Detach{})
}

func (Detach) Aliases() []string {
	return []string{"detach"}
}

func (*Detach) CompletePath(arg string) []string {
	composer, _ := app.SelectedTabContent().(*app.Composer)
	return commands.CompletionFromList(composer.GetAttachments(), arg)
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

	if err := composer.DeleteAttachment(d.Path); err != nil {
		return err
	}

	app.PushSuccess(fmt.Sprintf("Detached %s", d.Path))

	return nil
}
