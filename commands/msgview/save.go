package msgview

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib/log"
	"git.sr.ht/~rjarry/aerc/lib/xdg"
	"git.sr.ht/~rjarry/aerc/models"
)

type Save struct {
	Force          bool   `opt:"-f" desc:"Overwrite destination path."`
	CreateDirs     bool   `opt:"-p" desc:"Create missing directories."`
	Attachments    bool   `opt:"-a" desc:"Save all attachments parts."`
	AllAttachments bool   `opt:"-A" desc:"Save all named parts."`
	Path           string `opt:"path" required:"false" complete:"CompletePath" desc:"Target file path."`
}

func init() {
	commands.Register(Save{})
}

func (Save) Description() string {
	return "Save the current message part to the given path."
}

func (Save) Context() commands.CommandContext {
	return commands.MESSAGE_VIEWER
}

func (Save) Aliases() []string {
	return []string{"save"}
}

func (*Save) CompletePath(arg string) []string {
	defaultPath := config.General().DefaultSavePath
	if defaultPath != "" && !isAbsPath(arg) {
		arg = filepath.Join(defaultPath, arg)
	}
	return commands.CompletePath(arg, false)
}

func (s Save) Execute(args []string) error {
	// we either need a path or a defaultPath
	if s.Path == "" && config.General().DefaultSavePath == "" {
		return errors.New("No default save path in config")
	}

	// Absolute paths are taken as is so that the user can override the default
	// if they want to
	if !isAbsPath(s.Path) {
		s.Path = filepath.Join(config.General().DefaultSavePath, s.Path)
	}

	s.Path = xdg.ExpandHome(s.Path)

	mv, ok := app.SelectedTabContent().(*app.MessageViewer)
	if !ok {
		return fmt.Errorf("SelectedTabContent is not a MessageViewer")
	}

	if s.Attachments || s.AllAttachments {
		parts := mv.AttachmentParts(s.AllAttachments)
		if len(parts) == 0 {
			return fmt.Errorf("This message has no attachments")
		}
		names := make(map[string]struct{})
		for _, pi := range parts {
			if err := s.savePart(pi, mv, names); err != nil {
				return err
			}
		}
		return nil
	}

	pi := mv.SelectedMessagePart()
	return s.savePart(pi, mv, make(map[string]struct{}))
}

func (s *Save) savePart(
	pi *app.PartInfo,
	mv *app.MessageViewer,
	names map[string]struct{},
) error {
	path := s.Path
	if s.Attachments || s.AllAttachments || isDirExists(path) {
		filename := generateFilename(pi.Part)
		path = filepath.Join(path, filename)
	}

	dir := filepath.Dir(path)
	if s.CreateDirs && dir != "" {
		err := os.MkdirAll(dir, 0o755)
		if err != nil {
			return err
		}
	}

	path = getCollisionlessFilename(path, names)
	names[path] = struct{}{}

	if pathExists(path) && !s.Force {
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
