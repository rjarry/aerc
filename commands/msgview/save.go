package msgview

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"git.sr.ht/~sircmpwn/getopt"
	"github.com/mitchellh/go-homedir"

	"git.sr.ht/~sircmpwn/aerc/commands"
	"git.sr.ht/~sircmpwn/aerc/models"
	"git.sr.ht/~sircmpwn/aerc/widgets"
)

type Save struct{}

func init() {
	register(Save{})
}

func (Save) Aliases() []string {
	return []string{"save"}
}

func (Save) Complete(aerc *widgets.Aerc, args []string) []string {
	path := strings.Join(args, " ")
	return commands.CompletePath(path)
}

func (Save) Execute(aerc *widgets.Aerc, args []string) error {
	opts, optind, err := getopt.Getopts(args, "fp")
	if err != nil {
		return err
	}

	var (
		force         bool
		createDirs    bool
		trailingSlash bool
	)

	for _, opt := range opts {
		switch opt.Option {
		case 'f':
			force = true
		case 'p':
			createDirs = true
		}
	}

	defaultPath := aerc.Config().General.DefaultSavePath
	// we either need a path or a defaultPath
	if defaultPath == "" && len(args) == optind {
		return errors.New("Usage: :save [-fp] <path>")
	}

	// as a convenience we join with spaces, so that the user doesn't need to
	// quote filenames containing spaces
	path := strings.Join(args[optind:], " ")

	// needs to be determined prior to calling filepath.Clean / filepath.Join
	// it gets stripped by Clean.
	// we auto generate a name if a directory was given
	if len(path) > 0 {
		trailingSlash = path[len(path)-1] == '/'
	} else if len(defaultPath) > 0 && len(path) == 0 {
		// empty path, so we might have a default that ends in a trailingSlash
		trailingSlash = defaultPath[len(defaultPath)-1] == '/'
	}

	// Absolute paths are taken as is so that the user can override the default
	// if they want to
	if !isAbsPath(path) {
		path = filepath.Join(defaultPath, path)
	}

	path, err = homedir.Expand(path)
	if err != nil {
		return err
	}

	mv, ok := aerc.SelectedTab().(*widgets.MessageViewer)
	if !ok {
		return fmt.Errorf("SelectedTab is not a MessageViewer")
	}
	pi := mv.SelectedMessagePart()

	if trailingSlash || isDirExists(path) {
		filename := generateFilename(pi.Part)
		path = filepath.Join(path, filename)
	}

	dir := filepath.Dir(path)
	if createDirs && dir != "" {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			return err
		}
	}

	if pathExists(path) && !force {
		return fmt.Errorf("%q already exists and -f not given", path)
	}

	ch := make(chan error, 1)
	store := mv.Store()
	store.FetchBodyPart(pi.Msg.Uid, pi.Index, func(reader io.Reader) {
		f, err := os.Create(path)
		if err != nil {
			ch <- err
			return
		}
		defer f.Close()
		_, err = io.Copy(f, reader)
		if err != nil {
			ch <- err
			return
		}
		ch <- nil
	})

	// we need to wait for the callback prior to displaying a result
	go func() {
		err := <-ch
		if err != nil {
			aerc.PushError(fmt.Sprintf("Save failed: %v", err))
			return
		}
		aerc.PushStatus("Saved to "+path, 10*time.Second)
	}()
	return nil
}

//isDir returns true if path is a directory and exists
func isDirExists(path string) bool {
	pathinfo, err := os.Stat(path)
	if err != nil {
		return false // we don't really care
	}
	if pathinfo.IsDir() {
		return true
	}
	return false
}

//pathExists returns true if path exists
func pathExists(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		return false // we don't really care why it failed
	}
	return true
}

//isAbsPath returns true if path given is anchored to / or . or ~
func isAbsPath(path string) bool {
	if len(path) == 0 {
		return false
	}
	switch path[0] {
	case '/':
		return true
	case '.':
		return true
	case '~':
		return true
	default:
		return false
	}
}

// generateFilename tries to get the filename from the given part.
// if that fails it will fallback to a generated one based on the date
func generateFilename(part *models.BodyStructure) string {
	var filename string
	if fn, ok := part.DispositionParams["filename"]; ok {
		filename = fn
	} else if fn, ok := part.Params["name"]; ok {
		filename = fn
	} else {
		timestamp := time.Now().Format("2006-01-02-150405")
		filename = fmt.Sprintf("aerc_%v", timestamp)
	}
	return filename
}
