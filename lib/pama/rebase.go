package pama

import (
	"fmt"

	"git.sr.ht/~rjarry/aerc/lib/pama/models"
)

// RebaseCommits fetches the commits between baseID and HEAD. The tags from the
// current project will be mapped onto the fetched commits based on either the
// commit hash or the commit subject.
func (m PatchManager) RebaseCommits(p models.Project, baseID string) ([]models.Commit, error) {
	rc, err := m.rc(p.RevctrlID, p.Root)
	if err != nil {
		return nil, revErr(err)
	}

	if !rc.Exists(baseID) {
		return nil, fmt.Errorf("cannot rebase on %s. "+
			"commit does not exist", baseID)
	}

	commitIDs, err := rc.History(baseID)
	if err != nil {
		return nil, err
	}

	commits := make([]models.Commit, len(commitIDs))
	for i := range commitIDs {
		commits[i] = models.NewCommit(
			rc,
			commitIDs[i],
			models.Untracked,
		)
	}

	// map tags from the commits from the project p
	for i, r := range commits {
		for _, c := range p.Commits {
			if c.ID == r.ID || c.Subject == r.Subject {
				commits[i].MessageId = c.MessageId
				commits[i].Tag = c.Tag
				break
			}
		}
	}

	return commits, nil
}

// SaveRebased checks if the commits actually exist in the repo, repopulate the
// info fields and saves the baseID for project p.
func (m PatchManager) SaveRebased(p models.Project, baseID string, commits []models.Commit) error {
	rc, err := m.rc(p.RevctrlID, p.Root)
	if err != nil {
		return revErr(err)
	}

	exist := make([]models.Commit, 0, len(commits))
	for _, c := range commits {
		if !rc.Exists(c.ID) {
			continue
		}
		exist = append(exist, c)
	}

	for i, c := range exist {
		exist[i].Subject = rc.Subject(c.ID)
		exist[i].Author = rc.Author(c.ID)
		exist[i].Date = rc.Date(c.ID)
	}

	p.Commits = exist

	if rc.Exists(baseID) {
		p.Base = models.NewCommit(rc, baseID, "")
	}

	err = m.store().StoreProject(p, true)
	return storeErr(err)
}
