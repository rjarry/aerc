package patch

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib/log"
	"git.sr.ht/~rjarry/aerc/lib/pama"
	"git.sr.ht/~rjarry/aerc/lib/pama/models"
	"git.sr.ht/~rjarry/aerc/lib/ui"
)

type Rebase struct {
	Commit string `opt:"commit" required:"false"`
}

func init() {
	register(Rebase{})
}

func (Rebase) Context() commands.CommandContext {
	return commands.GLOBAL
}

func (Rebase) Aliases() []string {
	return []string{"rebase"}
}

func (r Rebase) Execute(args []string) error {
	m := pama.New()
	current, err := m.CurrentProject()
	if err != nil {
		return err
	}

	baseID := r.Commit
	if baseID == "" {
		baseID = current.Base.ID
	}

	commits, err := m.RebaseCommits(current, baseID)
	if err != nil {
		return err
	}

	if len(commits) == 0 {
		err := m.SaveRebased(current, baseID, nil)
		if err != nil {
			return fmt.Errorf("No commits to rebase, but saving of new reference failed: %w", err)
		}
		app.PushStatus("No commits to rebase.", 10*time.Second)
		return nil
	}

	rebase := newRebase(commits)
	f, err := os.CreateTemp("", "aerc-patch-rebase-*")
	if err != nil {
		return err
	}
	name := f.Name()
	_, err = io.Copy(f, rebase.content())
	if err != nil {
		return err
	}
	f.Close()

	createWidget := func() (ui.DrawableInteractive, error) {
		editorCmd, err := app.CmdFallbackSearch(config.EditorCmds(), true)
		if err != nil {
			return nil, err
		}
		editor := exec.Command("/bin/sh", "-c", editorCmd+" "+name)
		term, err := app.NewTerminal(editor)
		if err != nil {
			return nil, err
		}
		term.OnClose = func(_ error) {
			app.CloseDialog()
			defer os.Remove(name)
			defer term.Focus(false)

			f, err := os.Open(name)
			if err != nil {
				app.PushError(fmt.Sprintf("failed to open file: %v", err))
				return
			}
			defer f.Close()

			if editor.ProcessState.ExitCode() > 0 {
				app.PushError("Quitting rebase without saving.")
				return
			}
			err = m.SaveRebased(current, baseID, rebase.parse(f))
			if err != nil {
				app.PushError(fmt.Sprintf("Failed to save rebased commits: %v", err))
				return
			}
			app.PushStatus("Successfully rebased.", 10*time.Second)
		}
		term.Show(true)
		term.Focus(true)
		return term, nil
	}

	viewer, err := createWidget()
	if err != nil {
		return err
	}

	app.AddDialog(app.DefaultDialog(
		ui.NewBox(viewer, fmt.Sprintf("Patch Rebase on %-6.6s", baseID), "",
			app.SelectedAccountUiConfig(),
		),
	))

	return nil
}

type rebase struct {
	commits []models.Commit
	table   map[string]models.Commit
	order   []string
}

func newRebase(commits []models.Commit) *rebase {
	return &rebase{
		commits: commits,
		table:   make(map[string]models.Commit),
	}
}

const (
	header string = ""
	footer string = `
# Rebase aerc's patch data. This will not affect the underlying repository in
# any way.
#
# Change the name in the first column to assign a new tag to a commit. To group
# multiple commits, use the same tag name.
#
# An 'untracked' tag indicates that aerc lost track of that commit, either due
# to a commit-hash change or because that commit was applied outside of aerc.
#
# Do not change anything else besides the tag names (first column).
#
# Do not reorder the lines. The ordering should remain as in the repository.
#
# If you remove a line or keep an 'untracked' tag, those commits will be removed
# from aerc's patch tracking.
#
`
)

func (r *rebase) content() io.Reader {
	var buf bytes.Buffer
	buf.WriteString(header)
	for _, c := range r.commits {
		tag := c.Tag
		if tag == "" {
			tag = models.Untracked
		}
		shortHash := fmt.Sprintf("%6.6s", c.ID)
		buf.WriteString(
			fmt.Sprintf("%-12s     %6.6s     %s\n", tag, shortHash, c.Info()))
		r.table[shortHash] = c
		r.order = append(r.order, shortHash)
	}
	buf.WriteString(footer)
	return &buf
}

func (r *rebase) parse(reader io.Reader) []models.Commit {
	var commits []models.Commit
	var hashes []string
	scanner := bufio.NewScanner(reader)
	duplicated := make(map[string]struct{})
	for scanner.Scan() {
		s := scanner.Text()
		i := strings.Index(s, "#")
		if i >= 0 {
			s = s[:i]
		}
		if strings.TrimSpace(s) == "" {
			continue
		}

		fds := strings.Fields(s)
		if len(fds) < 2 {
			continue
		}

		tag, shortHash := fds[0], fds[1]
		if tag == models.Untracked {
			continue
		}
		_, dedup := duplicated[shortHash]
		if dedup {
			log.Warnf("rebase: skipping duplicated hash: %s", shortHash)
			continue
		}

		hashes = append(hashes, shortHash)
		c, ok := r.table[shortHash]
		if !ok {
			log.Errorf("Looks like the commit hashes were changed "+
				"during the rebase. Dropping: %v", shortHash)
			continue
		}
		log.Tracef("save commit %s with tag %s", shortHash, tag)
		c.Tag = tag
		commits = append(commits, c)
		duplicated[shortHash] = struct{}{}
	}
	reorder(commits, hashes, r.order)
	return commits
}

func reorder(toSort []models.Commit, now []string, by []string) {
	byMap := make(map[string]int)
	for i, s := range by {
		byMap[s] = i
	}

	complete := true
	for _, s := range now {
		_, ok := byMap[s]
		complete = complete && ok
	}
	if !complete {
		return
	}

	sort.SliceStable(toSort, func(i, j int) bool {
		return byMap[now[i]] < byMap[now[j]]
	})
}
