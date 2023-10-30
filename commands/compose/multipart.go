package compose

import (
	"bytes"
	"fmt"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/config"
)

type Multipart struct {
	Remove bool   `opt:"-d"`
	Mime   string `opt:"mime" metavar:"<mime/type>" complete:"CompleteMime"`
}

func init() {
	register(Multipart{})
}

func (Multipart) Aliases() []string {
	return []string{"multipart"}
}

func (*Multipart) CompleteMime(arg string) []string {
	var completions []string
	for mime := range config.Converters {
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
		_, found := config.Converters[m.Mime]
		if !found {
			return fmt.Errorf("no command defined for MIME type: %s", m.Mime)
		}
		err := composer.AppendPart(
			m.Mime,
			map[string]string{"Charset": "UTF-8"},
			// the actual content of the part will be rendered
			// every time the body of the email is updated
			bytes.NewReader([]byte{}),
		)
		if err != nil {
			return err
		}
	}

	return nil
}
