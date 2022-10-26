package lib

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"git.sr.ht/~rjarry/aerc/models"
	"github.com/emersion/go-maildir"
)

type MaildirStore struct {
	root      string
	maildirpp bool // whether to use Maildir++ directory layout
}

func NewMaildirStore(root string, maildirpp bool) (*MaildirStore, error) {
	f, err := os.Open(root)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	s, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !s.IsDir() {
		return nil, fmt.Errorf("Given maildir '%s' not a directory", root)
	}
	return &MaildirStore{
		root: root, maildirpp: maildirpp,
	}, nil
}

func (s *MaildirStore) FolderMap() (map[string]maildir.Dir, error) {
	folders := make(map[string]maildir.Dir)
	if s.maildirpp {
		// In Maildir++ layout, INBOX is the root folder
		folders["INBOX"] = maildir.Dir(s.root)
	}
	err := filepath.Walk(s.root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("Invalid path '%s': error: %w", path, err)
		}
		if !info.IsDir() {
			return nil
		}

		// Skip maildir's default directories
		n := info.Name()
		if n == "new" || n == "tmp" || n == "cur" {
			return filepath.SkipDir
		}

		// Get the relative path from the parent directory
		dirPath, err := filepath.Rel(s.root, path)
		if err != nil {
			return err
		}

		// Skip the parent directory
		if dirPath == "." {
			return nil
		}

		// Drop dirs that lack {new,tmp,cur} subdirs
		for _, sub := range []string{"new", "tmp", "cur"} {
			if _, err := os.Stat(filepath.Join(path, sub)); os.IsNotExist(err) {
				return nil
			}
		}

		if s.maildirpp {
			// In Maildir++ layout, mailboxes are stored in a single directory
			// and prefixed with a dot, and subfolders are separated by dots.
			if !strings.HasPrefix(dirPath, ".") {
				return filepath.SkipDir
			}
			dirPath = strings.TrimPrefix(dirPath, ".")
			dirPath = strings.ReplaceAll(dirPath, ".", "/")
			folders[dirPath] = maildir.Dir(path)

			// Since all mailboxes are stored in a single directory, don't
			// recurse into subdirectories
			return filepath.SkipDir
		}

		folders[dirPath] = maildir.Dir(path)
		return nil
	})
	return folders, err
}

// Folder returns a maildir.Dir with the specified name inside the Store
func (s *MaildirStore) Dir(name string) maildir.Dir {
	if s.maildirpp {
		// Use Maildir++ layout
		if name == "INBOX" {
			return maildir.Dir(s.root)
		}
		return maildir.Dir(filepath.Join(s.root, "."+strings.ReplaceAll(name, "/", ".")))
	}
	return maildir.Dir(filepath.Join(s.root, name))
}

// uidReg matches filename encoded UIDs in maildirs synched with mbsync or
// OfflineIMAP
var uidReg = regexp.MustCompile(`,U=\d+`)

func StripUIDFromMessageFilename(basename string) string {
	return uidReg.ReplaceAllString(basename, "")
}

var MaildirToFlag = map[maildir.Flag]models.Flag{
	maildir.FlagReplied: models.AnsweredFlag,
	maildir.FlagSeen:    models.SeenFlag,
	maildir.FlagTrashed: models.DeletedFlag,
	maildir.FlagFlagged: models.FlaggedFlag,
	// maildir.FlagDraft Flag = 'D'
	// maildir.FlagPassed Flag = 'P'
}

var FlagToMaildir = map[models.Flag]maildir.Flag{
	models.AnsweredFlag: maildir.FlagReplied,
	models.SeenFlag:     maildir.FlagSeen,
	models.DeletedFlag:  maildir.FlagTrashed,
	models.FlaggedFlag:  maildir.FlagFlagged,
	// maildir.FlagDraft Flag = 'D'
	// maildir.FlagPassed Flag = 'P'
}

func FromMaildirFlags(maildirFlags []maildir.Flag) []models.Flag {
	var flags []models.Flag
	for _, maildirFlag := range maildirFlags {
		if flag, ok := MaildirToFlag[maildirFlag]; ok {
			flags = append(flags, flag)
		}
	}
	return flags
}

func ToMaildirFlags(flags []models.Flag) []maildir.Flag {
	var maildirFlags []maildir.Flag
	for _, flag := range flags {
		if maildirFlag, ok := FlagToMaildir[flag]; ok {
			maildirFlags = append(maildirFlags, maildirFlag)
		}
	}
	return maildirFlags
}
