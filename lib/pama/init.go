package pama

import (
	"git.sr.ht/~rjarry/aerc/lib/pama/models"
)

// Init creates a new revision control project
func (m PatchManager) Init(name, path string, overwrite bool) error {
	id, root, err := m.detect(path)
	if err != nil {
		return err
	}
	rc, err := m.rc(id, root)
	if err != nil {
		return err
	}
	headID, err := rc.Head()
	if err != nil {
		return err
	}
	p := models.Project{
		Name:      name,
		Root:      root,
		RevctrlID: id,
		Base:      models.NewCommit(rc, headID, ""),
		Commits:   make([]models.Commit, 0),
	}
	store := m.store()
	err = store.StoreProject(p, overwrite)
	if err != nil {
		return storeErr(err)
	}
	return storeErr(store.SetCurrent(name))
}
