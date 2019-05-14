package compose

import (
	"errors"
	"os"

	"git.sr.ht/~sircmpwn/aerc2/widgets"
)

func init() {
	register("send-message", SendMessage)
}

func SendMessage(aerc *widgets.Aerc, args []string) error {
	if len(args) > 1 {
		return errors.New("Usage: send-message")
	}
	composer, _ := aerc.SelectedTab().(*widgets.Composer)
	//config := composer.Config()
	f, err := os.Create("/tmp/test.eml")
	if err != nil {
		panic(err)
	}
	_, err = composer.Message(f)
	if err != nil {
		panic(err)
	}
	return nil
}
