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

	"github.com/lithammer/fuzzysearch/fuzzy"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/lib"
	"git.sr.ht/~rjarry/aerc/lib/log"
	"git.sr.ht/~rjarry/aerc/lib/xdg"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/types"
	"git.sr.ht/~rjarry/go-opt"
	"git.sr.ht/~rockorager/vaxis"
)

// QuickTerm is an ephemeral terminal for running a single command and quitting.
func QuickTerm(args []string, stdin io.Reader) (*app.Terminal, error) {
	cmd := exec.Command(args[0], args[1:]...)
	pipe, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	term, err := app.NewTerminal(cmd)
	if err != nil {
		return nil, err
	}

	term.OnClose = func(err error) {
		if err != nil {
			app.PushError(err.Error())
			// remove the tab on error, otherwise it gets stuck
			app.RemoveTab(term, false)
		} else {
			app.PushStatus("Process complete, press any key to close.",
				10*time.Second)
			term.OnEvent = func(event vaxis.Event) bool {
				app.RemoveTab(term, true)
				return true
			}
		}
	}

	term.OnStart = func() {
		status := make(chan error, 1)

		go func() {
			defer log.PanicHandler()

			_, err := io.Copy(pipe, stdin)
			defer pipe.Close()
			status <- err
		}()

		err := <-status
		if err != nil {
			app.PushError(err.Error())
		}
	}

	return term, nil
}

// CompletePath provides filesystem completions given a starting path.
func CompletePath(path string, onlyDirs bool) []string {
	if path == ".." || strings.HasSuffix(path, "/..") {
		return []string{path + "/"}
	}
	if path == "~" || strings.HasPrefix(path, "~/") {
		path = xdg.HomeDir() + strings.TrimPrefix(path, "~")
	}
	includeHidden := path == "."
	if i := strings.LastIndex(path, "/"); i != -1 && i < len(path)-1 {
		includeHidden = strings.HasPrefix(path[i+1:], ".")
	}

	matches, err := filepath.Glob(path + "*")
	if err != nil || matches == nil {
		return nil
	}

	results := make([]string, 0, len(matches))

	for _, m := range matches {
		if isDir(m) {
			m += "/"
		} else if onlyDirs {
			continue
		}
		if strings.HasPrefix(filepath.Base(m), ".") && !includeHidden {
			continue
		}
		results = append(results, opt.QuoteArg(xdg.TildeHome(m)))
	}

	sort.Strings(results)

	return results
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
func MarkedOrSelected(pm app.ProvidesMessages) ([]uint32, error) {
	// marked has priority over the selected message
	marked, err := pm.MarkedMessages()
	if err != nil {
		return nil, err
	}
	if len(marked) > 0 {
		marked = expandFoldedThreads(pm, marked)
		return marked, nil
	}
	msg, err := pm.SelectedMessage()
	if err != nil {
		return nil, err
	}
	return expandFoldedThreads(pm, []uint32{msg.Uid}), nil
}

func expandFoldedThreads(pm app.ProvidesMessages, uids []uint32) []uint32 {
	store := pm.Store()
	if store == nil {
		return uids
	}
	expanded := make([]uint32, len(uids))
	copy(expanded, uids)
	for _, uid := range uids {
		thread, err := store.Thread(uid)
		if err != nil {
			continue
		}
		if thread != nil && thread.FirstChild != nil && thread.FirstChild.Hidden > 0 {
			_ = thread.Walk(func(t *types.Thread, _ int, __ error) error {
				if t.Uid != uid {
					expanded = append(expanded, t.Uid)
				}
				return nil
			})
		}

	}
	if len(uids) != len(expanded) {
		log.Debugf("expand folded threads: %v -> %v\n", uids, expanded)
	}
	return expanded
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

func MsgInfoFromUids(store *lib.MessageStore, uids []uint32, statusInfo func(string)) ([]*models.MessageInfo, error) {
	infos := make([]*models.MessageInfo, len(uids))
	needHeaders := make([]uint32, 0)
	for i, uid := range uids {
		var ok bool
		infos[i], ok = store.Messages[uid]
		if !ok {
			return nil, fmt.Errorf("uid not found")
		}
		if infos[i] == nil {
			needHeaders = append(needHeaders, uid)
		}
	}
	if len(needHeaders) > 0 {
		store.FetchHeaders(needHeaders, func(msg types.WorkerMessage) {
			var info string
			switch m := msg.(type) {
			case *types.Done:
				info = "All headers fetched. Please repeat command."
			case *types.Error:
				info = fmt.Sprintf("Encountered error while fetching headers: %v", m.Error)
			}
			if statusInfo != nil {
				statusInfo(info)
			}
		})
		return nil, fmt.Errorf("Fetching missing message headers. Please wait.")
	}
	return infos, nil
}

func QuoteSpace(s string) string {
	return opt.QuoteArg(s) + " "
}

// FilterList takes a list of valid completions and filters it, either
// by case smart prefix, or by fuzzy matching
// An optional post processing function can be passed to prepend, append or
// quote each value.
func FilterList(
	valid []string, search string, postProc func(string) string,
) []string {
	if postProc == nil {
		postProc = opt.QuoteArg
	}
	out := make([]string, 0, len(valid))
	if app.SelectedAccountUiConfig().FuzzyComplete {
		for _, v := range fuzzy.RankFindFold(search, valid) {
			out = append(out, postProc(v.Target))
		}
	} else {
		for _, v := range valid {
			if hasCaseSmartPrefix(v, search) {
				out = append(out, postProc(v))
			}
		}
	}
	return out
}
