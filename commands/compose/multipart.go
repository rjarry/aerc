package compose

import (
	"bytes"
	"fmt"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~sircmpwn/getopt"
)

type Multipart struct{}

func init() {
	register(Multipart{})
}

func (Multipart) Aliases() []string {
	return []string{"multipart"}
}

func (Multipart) Complete(aerc *app.Aerc, args []string) []string {
	var completions []string
	completions = append(completions, "-d")
	for mime := range config.Converters {
		completions = append(completions, mime)
	}
	return commands.CompletionFromList(aerc, completions, args)
}

func (a Multipart) Execute(aerc *app.Aerc, args []string) error {
	composer, ok := aerc.SelectedTabContent().(*app.Composer)
	if !ok {
		return fmt.Errorf(":multipart is only available on the compose::review screen")
	}

	opts, optind, err := getopt.Getopts(args, "d")
	if err != nil {
		return fmt.Errorf("Usage: :multipart [-d] <mime/type>")
	}
	var remove bool = false
	for _, opt := range opts {
		if opt.Option == 'd' {
			remove = true
		}
	}
	args = args[optind:]
	if len(args) != 1 {
		return fmt.Errorf("Usage: :multipart [-d] <mime/type>")
	}
	mime := args[0]

	if remove {
		return composer.RemovePart(mime)
	} else {
		_, found := config.Converters[mime]
		if !found {
			return fmt.Errorf("no command defined for MIME type: %s", mime)
		}
		err = composer.AppendPart(
			mime,
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
