package msgview

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime/quotedprintable"
	"os"
	"path/filepath"
	"strings"
	"time"

	"git.sr.ht/~sircmpwn/aerc/widgets"
	"git.sr.ht/~sircmpwn/getopt"
	"github.com/mitchellh/go-homedir"
)

type Save struct{}

func init() {
	register(Save{})
}

func (_ Save) Aliases() []string {
	return []string{"save"}
}

func (_ Save) Complete(aerc *widgets.Aerc, args []string) []string {
	return nil
}

func (_ Save) Execute(aerc *widgets.Aerc, args []string) error {
	opts, optind, err := getopt.Getopts(args, "p")
	if err != nil {
		return err
	}

	var (
		mkdirs bool
		path   string
	)

	for _, opt := range opts {
		switch opt.Option {
		case 'p':
			mkdirs = true
		}
	}
	if len(args) == optind+1 {
		path = args[optind]
	} else if defaultPath := aerc.Config().General.DefaultSavePath; defaultPath != "" {
		path = defaultPath
	} else {
		return errors.New("Usage: :save [-p] <path>")
	}

	mv := aerc.SelectedTab().(*widgets.MessageViewer)
	p := mv.SelectedMessagePart()

	p.Store.FetchBodyPart(p.Msg.Uid, p.Index, func(reader io.Reader) {
		// email parts are encoded as 7bit (plaintext), quoted-printable, or base64

		if strings.EqualFold(p.Part.Encoding, "base64") {
			reader = base64.NewDecoder(base64.StdEncoding, reader)
		} else if strings.EqualFold(p.Part.Encoding, "quoted-printable") {
			reader = quotedprintable.NewReader(reader)
		}

		var pathIsDir bool
		if path[len(path)-1:] == "/" {
			pathIsDir = true
		}
		// Note: path expansion has to happen after test for trailing /,
		// since it is stripped when path is expanded
		path, err := homedir.Expand(path)
		if err != nil {
			aerc.PushError(" " + err.Error())
		}

		pathinfo, err := os.Stat(path)
		if err == nil && pathinfo.IsDir() {
			pathIsDir = true
		} else if os.IsExist(err) && pathIsDir {
			aerc.PushError("The given directory is an existing file")
		}
		var (
			save_file string
			save_dir  string
		)
		if pathIsDir {
			save_dir = path
			if filename, ok := p.Part.DispositionParams["filename"]; ok {
				save_file = filename
			} else {
				timestamp := time.Now().Format("2006-01-02-150405")
				save_file = fmt.Sprintf("aerc_%v", timestamp)
			}
		} else {
			save_file = filepath.Base(path)
			save_dir = filepath.Dir(path)
		}
		if _, err := os.Stat(save_dir); os.IsNotExist(err) {
			if mkdirs {
				os.MkdirAll(save_dir, 0755)
			} else {
				aerc.PushError("Target directory does not exist, use " +
					":save with the -p option to create it")
				return
			}
		}
		target := filepath.Clean(filepath.Join(save_dir, save_file))

		f, err := os.Create(target)
		if err != nil {
			aerc.PushError(" " + err.Error())
			return
		}
		defer f.Close()

		_, err = io.Copy(f, reader)
		if err != nil {
			aerc.PushError(" " + err.Error())
			return
		}

		aerc.PushStatus("Saved to "+target, 10*time.Second)
	})

	return nil
}
