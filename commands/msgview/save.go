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

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib/xdg"
	"git.sr.ht/~rjarry/aerc/log"
	"git.sr.ht/~rjarry/aerc/models"
)

type Save struct{}

func init() {
	register(Save{})
}

func (Save) Options() string {
	return "fpaA"
}

func (Save) Aliases() []string {
	return []string{"save"}
}

func (s Save) Complete(args []string) []string {
	trimmed := commands.Operands(args, s.Options())
	path := strings.Join(trimmed, " ")
	defaultPath := config.General.DefaultSavePath
	if defaultPath != "" && !isAbsPath(path) {
		path = filepath.Join(defaultPath, path)
	}
	return commands.CompletePath(xdg.ExpandHome(path))
}

type saveParams struct {
	force          bool
	createDirs     bool
	trailingSlash  bool
	attachments    bool
	allAttachments bool
}

func (s Save) Execute(args []string) error {
	opts, optind, err := getopt.Getopts(args, s.Options())
	if err != nil {
		return err
	}

	var params saveParams

	for _, opt := range opts {
		switch opt.Option {
		case 'f':
			params.force = true
		case 'p':
			params.createDirs = true
		case 'a':
			params.attachments = true
		case 'A':
			params.allAttachments = true
		}
	}

	defaultPath := config.General.DefaultSavePath
	// we either need a path or a defaultPath
	if defaultPath == "" && len(args) == optind {
		return errors.New("Usage: :save [-fpa] <path>")
	}

	// as a convenience we join with spaces, so that the user doesn't need to
	// quote filenames containing spaces
	path := strings.Join(args[optind:], " ")

	// needs to be determined prior to calling filepath.Clean / filepath.Join
	// it gets stripped by Clean.
	// we auto generate a name if a directory was given
	if len(path) > 0 {
		params.trailingSlash = path[len(path)-1] == '/'
	} else if len(defaultPath) > 0 && len(path) == 0 {
		// empty path, so we might have a default that ends in a trailingSlash
		params.trailingSlash = defaultPath[len(defaultPath)-1] == '/'
	}

	// Absolute paths are taken as is so that the user can override the default
	// if they want to
	if !isAbsPath(path) {
		path = filepath.Join(defaultPath, path)
	}

	path = xdg.ExpandHome(path)

	mv, ok := app.SelectedTabContent().(*app.MessageViewer)
	if !ok {
		return fmt.Errorf("SelectedTabContent is not a MessageViewer")
	}

	if params.attachments || params.allAttachments {
		parts := mv.AttachmentParts(params.allAttachments)
		if len(parts) == 0 {
			return fmt.Errorf("This message has no attachments")
		}
		params.trailingSlash = true
		names := make(map[string]struct{})
		for _, pi := range parts {
			if err := savePart(pi, path, mv, &params, names); err != nil {
				return err
			}
		}
		return nil
	}

	pi := mv.SelectedMessagePart()
	return savePart(pi, path, mv, &params, make(map[string]struct{}))
}

func savePart(
	pi *app.PartInfo,
	path string,
	mv *app.MessageViewer,
	params *saveParams,
	names map[string]struct{},
) error {
	if params.trailingSlash || isDirExists(path) {
		filename := generateFilename(pi.Part)
		path = filepath.Join(path, filename)
	}

	dir := filepath.Dir(path)
	if params.createDirs && dir != "" {
		err := os.MkdirAll(dir, 0o755)
		if err != nil {
			return err
		}
	}

	path = getCollisionlessFilename(path, names)
	names[path] = struct{}{}

	if pathExists(path) && !params.force {
		return fmt.Errorf("%q already exists and -f not given", path)
	}

	ch := make(chan error, 1)
	mv.MessageView().FetchBodyPart(pi.Index, func(reader io.Reader) {
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
		defer log.PanicHandler()

		err := <-ch
		if err != nil {
			app.PushError(fmt.Sprintf("Save failed: %v", err))
			return
		}
		app.PushStatus("Saved to "+path, 10*time.Second)
	}()
	return nil
}

func getCollisionlessFilename(path string, existing map[string]struct{}) string {
	ext := filepath.Ext(path)
	name := strings.TrimSuffix(path, ext)
	_, exists := existing[path]
	counter := 1
	for exists {
		path = fmt.Sprintf("%s_%d%s", name, counter, ext)
		counter++
		_, exists = existing[path]
	}
	return path
}

// isDir returns true if path is a directory and exists
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

// pathExists returns true if path exists
func pathExists(path string) bool {
	_, err := os.Stat(path)

	return err == nil
}

// isAbsPath returns true if path given is anchored to / or . or ~
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
	filename := part.FileName()
	// Some MUAs send attachments with names like /some/stupid/idea/happy.jpeg
	// Assuming non hostile intent it does make sense to use just the last
	// portion of the pathname as the filename for saving it.
	filename = filename[strings.LastIndex(filename, "/")+1:]
	switch filename {
	case "", ".", "..":
		timestamp := time.Now().Format("2006-01-02-150405")
		filename = fmt.Sprintf("aerc_%v", timestamp)
	default:
		// already have a valid name
	}
	return filename
}
