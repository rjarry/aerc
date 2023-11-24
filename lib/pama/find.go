package pama

import (
	"fmt"

	"git.sr.ht/~rjarry/aerc/lib/pama/models"
)

func (m PatchManager) Find(hash string, p models.Project) (models.Commit, error) {
	var c models.Commit
	rc, err := m.rc(p.RevctrlID, p.Root)
	if err != nil {
		return c, revErr(err)
	}
	if !rc.Exists(hash) {
		return c, fmt.Errorf("no commit found for hash %s", hash)
	}
	return models.NewCommit(rc, hash, ""), nil
}
