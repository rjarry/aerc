package compose

import (
	"fmt"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/config"
)

type Multipart struct {
	Remove bool   `opt:"-d" desc:"Remove the specified mime/type."`
	Mime   string `opt:"mime" metavar:"<mime/type>" complete:"CompleteMime" desc:"MIME/type name."`
}

func init() {
	commands.Register(Multipart{})
}

func (Multipart) Description() string {
	return "Convert the message to multipart with the given mime/type part."
}

func (Multipart) Context() commands.CommandContext {
	return commands.COMPOSE_EDIT | commands.COMPOSE_REVIEW
}

func (Multipart) Aliases() []string {
	return []string{"multipart"}
}

func (*Multipart) CompleteMime(arg string) []string {
	var completions []string
	for mime := range config.Converters() {
		completions = append(completions, mime)
	}
	return commands.FilterList(completions, arg, nil)
}

func (m Multipart) Execute(args []string) error {
	composer, ok := app.SelectedTabContent().(*app.Composer)
	if !ok {
		return fmt.Errorf(":multipart is only available on the compose::review screen")
	}

	if m.Remove {
		return composer.RemovePart(m.Mime)
	} else {
		_, found := config.Converters()[m.Mime]
		if !found {
			return fmt.Errorf("no command defined for MIME type: %s", m.Mime)
		}
		err := composer.AppendPart(
			m.Mime,
			map[string]string{"Charset": "UTF-8"},
			// the actual content of the part will be rendered
			// every time the body of the email is updated
			nil,
		)
		if err != nil {
			return err
		}
	}

	return nil
}
