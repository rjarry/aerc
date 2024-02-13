package pama

import (
	"fmt"

	"git.sr.ht/~rjarry/aerc/lib/log"
)

// Unlink removes provided project
func (m PatchManager) Unlink(name string) error {
	store := m.store()
	names, err := m.Names()
	if err != nil {
		return err
	}

	index := -1
	for i, s := range names {
		if s == name {
			index = i
			break
		}
	}
	if index < 0 {
		return fmt.Errorf("Project '%s' not found", name)
	}

	cur, err := store.CurrentName()
	if err == nil && cur == name {
		var next string
		for _, s := range names {
			if name != s {
				next = s
				break
			}
		}
		err = store.SetCurrent(next)
		if err != nil {
			return storeErr(err)
		}
	}

	p, err := store.Project(name)
	if err == nil && isWorktree(p) {
		err = m.deleteWorktree(p)
		if err != nil {
			log.Errorf("failed to delete worktree: %v", err)
		}
		err = store.SetCurrent(p.Worktree.Name)
		if err != nil {
			log.Errorf("failed to set current project: %v", err)
		}
	}

	return storeErr(m.store().DeleteProject(name))
}

func (m PatchManager) Names() ([]string, error) {
	names, err := m.store().Names()
	return names, storeErr(err)
}
