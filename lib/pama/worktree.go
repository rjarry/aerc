package pama

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"git.sr.ht/~rjarry/aerc/lib/log"
	"git.sr.ht/~rjarry/aerc/lib/pama/models"
	"git.sr.ht/~rjarry/aerc/lib/xdg"
)

func cacheDir() (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		dir = xdg.ExpandHome("~/.cache")
	}
	return path.Join(dir, "aerc"), nil
}

func makeWorktreeName(baseProject, tag string) string {
	unique, err := generateTag(4)
	if err != nil {
		log.Infof("could not generate unique id: %v", err)
	}
	return strings.Join([]string{baseProject, "worktree", tag, unique}, "_")
}

func isWorktree(p models.Project) bool {
	return p.Worktree.Name != "" && p.Worktree.Root != ""
}

func (m PatchManager) CreateWorktree(p models.Project, commitID, tag string,
) (models.Project, error) {
	var w models.Project

	if isWorktree(p) {
		return w, fmt.Errorf("This is already a worktree.")
	}

	w.RevctrlID = p.RevctrlID
	w.Base = models.Commit{ID: commitID}
	w.Name = makeWorktreeName(p.Name, tag)
	w.Worktree = models.WorktreeParent{Name: p.Name, Root: p.Root}

	dir, err := cacheDir()
	if err != nil {
		return p, err
	}
	w.Root = filepath.Join(dir, "worktrees", w.Name)

	rc, err := m.rc(p.RevctrlID, p.Root)
	if err != nil {
		return p, revErr(err)
	}

	err = rc.CreateWorktree(w.Root, w.Base.ID)
	if err != nil {
		return p, revErr(err)
	}

	err = m.store().StoreProject(w, true)
	if err != nil {
		return p, storeErr(err)
	}

	return w, nil
}

func (m PatchManager) deleteWorktree(p models.Project) error {
	if !isWorktree(p) {
		return nil
	}

	rc, err := m.rc(p.RevctrlID, p.Worktree.Root)
	if err != nil {
		return revErr(err)
	}

	err = rc.DeleteWorktree(p.Root)
	if err != nil {
		return revErr(err)
	}

	return nil
}
