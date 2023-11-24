package pama

import (
	"fmt"

	"git.sr.ht/~rjarry/aerc/lib/pama/models"
	"git.sr.ht/~rjarry/aerc/log"
)

func (m PatchManager) RemovePatch(patch string) error {
	p, err := m.CurrentProject()
	if err != nil {
		return err
	}

	if !models.Commits(p.Commits).HasTag(patch) {
		return fmt.Errorf("Patch '%s' not found in project '%s'", patch, p.Name)
	}

	rc, err := m.rc(p.RevctrlID, p.Root)
	if err != nil {
		return revErr(err)
	}

	if !rc.Clean() {
		return fmt.Errorf("Aborting... There are unstaged changes " +
			"or a rebase in progress")
	}

	toRemove := make([]models.Commit, 0)
	for _, c := range p.Commits {
		if !rc.Exists(c.ID) {
			log.Errorf("failed to find commit. %v", c)
			return fmt.Errorf("Cannot remove patch. " +
				"Please rebase first with ':patch rebase'")
		}
		if c.Tag == patch {
			toRemove = append(toRemove, c)
		}
	}

	removed := make(map[string]struct{})
	for i := len(toRemove) - 1; i >= 0; i-- {
		commitID := toRemove[i].ID
		beforeIDs, err := rc.History(commitID)
		if err != nil {
			log.Errorf("failed to remove %v (commits before): %v", toRemove[i], err)
			continue
		}
		err = rc.Remove(commitID)
		if err != nil {
			log.Errorf("failed to remove %v (remove): %v", toRemove[i], err)
			continue
		}
		removed[commitID] = struct{}{}
		afterIDs, err := rc.History(p.Base.ID)
		if err != nil {
			log.Errorf("failed to remove %v (commits after): %v", toRemove[i], err)
			continue
		}
		afterIDs = afterIDs[len(afterIDs)-len(beforeIDs):]
		transform := make(map[string]string)
		for j := 0; j < len(beforeIDs); j++ {
			transform[beforeIDs[j]] = afterIDs[j]
		}
		for j, c := range p.Commits {
			if newId, ok := transform[c.ID]; ok {
				msgid := p.Commits[j].MessageId
				p.Commits[j] = models.NewCommit(
					rc,
					newId,
					p.Commits[j].Tag,
				)
				p.Commits[j].MessageId = msgid
			}
		}
	}

	if len(removed) < len(toRemove) {
		return fmt.Errorf("Failed to remove commits. Removed %d of %d.",
			len(removed), len(toRemove))
	}

	commits := make([]models.Commit, 0, len(p.Commits))
	for _, c := range p.Commits {
		if _, ok := removed[c.ID]; ok {
			continue
		}
		commits = append(commits, c)
	}
	p.Commits = commits

	return storeErr(m.store().StoreProject(p, true))
}
