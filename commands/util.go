package commands

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"git.sr.ht/~sircmpwn/aerc/lib"
	"git.sr.ht/~sircmpwn/aerc/models"
	"git.sr.ht/~sircmpwn/aerc/widgets"
	"github.com/gdamore/tcell/v2"
	"github.com/mitchellh/go-homedir"
)

// QuickTerm is an ephemeral terminal for running a single command and quitting.
func QuickTerm(aerc *widgets.Aerc, args []string, stdin io.Reader) (*widgets.Terminal, error) {
	cmd := exec.Command(args[0], args[1:]...)
	pipe, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	term, err := widgets.NewTerminal(cmd)
	if err != nil {
		return nil, err
	}

	term.OnClose = func(err error) {
		if err != nil {
			aerc.PushError(" " + err.Error())
			// remove the tab on error, otherwise it gets stuck
			aerc.RemoveTab(term)
		} else {
			aerc.PushStatus("Process complete, press any key to close.",
				10*time.Second)
			term.OnEvent = func(event tcell.Event) bool {
				aerc.RemoveTab(term)
				return true
			}
		}
	}

	term.OnStart = func() {
		status := make(chan error, 1)

		go func() {
			_, err := io.Copy(pipe, stdin)
			defer pipe.Close()
			status <- err
		}()

		err := <-status
		if err != nil {
			aerc.PushError(" " + err.Error())
		}
	}

	return term, nil
}

// CompletePath provides filesystem completions given a starting path.
func CompletePath(path string) []string {
	if path == "" {
		// default to cwd
		cwd, err := os.Getwd()
		if err != nil {
			return nil
		}
		path = cwd
	}

	path, err := homedir.Expand(path)
	if err != nil {
		return nil
	}

	// strip trailing slashes, etc.
	path = filepath.Clean(path)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		// if the path doesn't exist, it is likely due to it being a partial path
		// in this case, we want to return possible matches (ie /hom* should match
		// /home)
		matches, err := filepath.Glob(fmt.Sprintf("%s*", path))
		if err != nil {
			return nil
		}

		for i, m := range matches {
			if isDir(m) {
				matches[i] = m + "/"
			}
		}

		sort.Strings(matches)
		return matches
	}

	files := listDir(path, false)

	for i, f := range files {
		f = filepath.Join(path, f)
		if isDir(f) {
			f += "/"
		}

		files[i] = f
	}

	sort.Strings(files)
	return files
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	return info.IsDir()
}

// return all filenames in a directory, optionally including hidden files
func listDir(path string, hidden bool) []string {
	f, err := os.Open(path)
	if err != nil {
		return []string{}
	}

	files, err := f.Readdirnames(-1) // read all dir names
	if err != nil {
		return []string{}
	}

	if hidden {
		return files
	}

	var filtered []string
	for _, g := range files {
		if !strings.HasPrefix(g, ".") {
			filtered = append(filtered, g)
		}
	}

	return filtered
}

// MarkedOrSelected returns either all marked messages if any are marked or the
// selected message instead
func MarkedOrSelected(pm widgets.ProvidesMessages) ([]uint32, error) {
	// marked has priority over the selected message
	marked, err := pm.MarkedMessages()
	if err != nil {
		return nil, err
	}
	if len(marked) > 0 {
		return marked, nil
	}
	msg, err := pm.SelectedMessage()
	if err != nil {
		return nil, err
	}
	return []uint32{msg.Uid}, nil
}

// UidsFromMessageInfos extracts a uid slice from a slice of MessageInfos
func UidsFromMessageInfos(msgs []*models.MessageInfo) []uint32 {
	uids := make([]uint32, len(msgs))
	i := 0
	for _, msg := range msgs {
		uids[i] = msg.Uid
		i++
	}
	return uids
}

func MsgInfoFromUids(store *lib.MessageStore, uids []uint32) ([]*models.MessageInfo, error) {
	infos := make([]*models.MessageInfo, len(uids))
	for i, uid := range uids {
		var ok bool
		infos[i], ok = store.Messages[uid]
		if !ok {
			return nil, fmt.Errorf("uid not found")
		}
	}
	return infos, nil
}
