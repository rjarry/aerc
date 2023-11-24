package revctrl

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"git.sr.ht/~rjarry/aerc/lib/pama/models"
	"git.sr.ht/~rjarry/aerc/log"
)

func init() {
	register("git", newGit)
}

func newGit(s string) models.RevisionController {
	return &git{path: strings.TrimSpace(s)}
}

type git struct {
	path string
}

func (g git) Support() bool {
	_, exitcode, err := g.do("rev-parse")
	return exitcode == 0 && err == nil
}

func (g git) Root() (string, error) {
	s, _, err := g.do("rev-parse", "--show-toplevel")
	return s, err
}

func (g git) Head() (string, error) {
	s, _, err := g.do("rev-list", "-n 1", "HEAD")
	return s, err
}

func (g git) History(commit string) ([]string, error) {
	s, _, err := g.do("rev-list", "--reverse", fmt.Sprintf("%s..HEAD", commit))
	return strings.Fields(s), err
}

func (g git) Subject(commit string) string {
	s, exitcode, err := g.do("log", "-1", "--pretty=%s", commit)
	if exitcode > 0 || err != nil {
		return ""
	}
	return s
}

func (g git) Author(commit string) string {
	s, exitcode, err := g.do("log", "-1", "--pretty=%an", commit)
	if exitcode > 0 || err != nil {
		return ""
	}
	return s
}

func (g git) Date(commit string) string {
	s, exitcode, err := g.do("log", "-1", "--pretty=%as", commit)
	if exitcode > 0 || err != nil {
		return ""
	}
	return s
}

func (g git) Remove(commit string) error {
	_, exitcode, err := g.do("rebase", "--onto", commit+"^", commit)
	if exitcode > 0 {
		return fmt.Errorf("failed to remove commit %s", commit)
	}
	return err
}

func (g git) Exists(commit string) bool {
	_, exitcode, err := g.do("merge-base", "--is-ancestor", commit, "HEAD")
	return exitcode == 0 && err == nil
}

func (g git) Clean() bool {
	// is a rebase in progress?
	dirs := []string{"rebase-merge", "rebase-apply"}
	for _, dir := range dirs {
		relPath, _, err := g.do("rev-parse", "--git-path", dir)
		if err == nil {
			if _, err := os.Stat(filepath.Join(g.path, relPath)); !os.IsNotExist(err) {
				log.Errorf("%s exists: another rebase in progress..", dir)
				return false
			}
		}
	}
	// are there unstaged changes?
	s, exitcode, err := g.do("diff-index", "HEAD")
	return len(s) == 0 && exitcode == 0 && err == nil
}

func (g git) ApplyCmd() string {
	// TODO: should we return a *exec.Cmd instead of a string?
	return fmt.Sprintf("git -C %s am -3 --empty drop", g.path)
}

func (g git) do(args ...string) (string, int, error) {
	proc := exec.Command("git", "-C", g.path)
	proc.Args = append(proc.Args, args...)
	proc.Env = os.Environ()
	result, err := proc.Output()
	return string(bytes.TrimSpace(result)), proc.ProcessState.ExitCode(), err
}
