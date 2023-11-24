package pama

import (
	"encoding/base64"
	"fmt"
	"math/rand"
	"strings"

	"git.sr.ht/~rjarry/aerc/lib/pama/models"
	"git.sr.ht/~rjarry/aerc/log"
)

func (m PatchManager) CurrentProject() (p models.Project, err error) {
	store := m.store()
	name, err := store.CurrentName()
	if name == "" || err != nil {
		log.Errorf("failed to get current name: %v", storeErr(err))
		err = fmt.Errorf("no current project set. " +
			"Run :patch init first")
		return
	}
	names, err := store.Names()
	if err != nil {
		err = storeErr(err)
		return
	}
	notFound := true
	for _, s := range names {
		if s == name {
			notFound = !notFound
			break
		}
	}
	if notFound {
		err = fmt.Errorf("project '%s' does not exist anymore. "+
			"Run :patch init or :patch switch", name)
		return
	}
	p, err = store.Current()
	if err != nil {
		err = storeErr(err)
	}
	return
}

func (m PatchManager) CurrentPatches() ([]string, error) {
	c, err := m.CurrentProject()
	if err != nil {
		return nil, err
	}
	return models.Commits(c.Commits).Tags(), nil
}

func (m PatchManager) Head(p models.Project) (string, error) {
	rc, err := m.rc(p.RevctrlID, p.Root)
	if err != nil {
		return "", revErr(err)
	}
	return rc.Head()
}

func (m PatchManager) Clean(p models.Project) bool {
	rc, err := m.rc(p.RevctrlID, p.Root)
	if err != nil {
		log.Errorf("could not get revctl: %v", revErr(err))
		return false
	}
	return rc.Clean()
}

func (m PatchManager) ApplyCmd(p models.Project) (string, error) {
	rc, err := m.rc(p.RevctrlID, p.Root)
	if err != nil {
		return "", revErr(err)
	}
	return rc.ApplyCmd(), nil
}

func generateTag(n int) (string, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func makeUnique(s string) string {
	tag, err := generateTag(4)
	if err != nil {
		return fmt.Sprintf("%s_%d", s, rand.Uint32())
	}
	return fmt.Sprintf("%s_%s", s, tag)
}

// ApplyUpdate is called after the commits have been applied with the
// ApplyCmd(). It will determine the additional commits from the commitID (last
// HEAD position), assign the patch tag to those commits and store them in
// project p.
func (m PatchManager) ApplyUpdate(p models.Project, patch, commitID string,
	kv map[string]string,
) (models.Project, error) {
	rc, err := m.rc(p.RevctrlID, p.Root)
	if err != nil {
		return p, revErr(err)
	}

	commitIDs, err := rc.History(commitID)
	if err != nil {
		return p, revErr(err)
	}
	if len(commitIDs) == 0 {
		return p, fmt.Errorf("no commits found for patch %s", patch)
	}

	if models.Commits(p.Commits).HasTag(patch) {
		log.Warnf("Patch name '%s' already exists", patch)
		patch = makeUnique(patch)
		log.Warnf("Creating new name: '%s'", patch)
	}

	for _, c := range commitIDs {
		nc := models.NewCommit(rc, c, patch)
		for msgid, subj := range kv {
			if nc.Subject == "" {
				continue
			}
			if strings.Contains(subj, nc.Subject) {
				nc.MessageId = msgid
			}
		}
		p.Commits = append(p.Commits, nc)
	}

	err = m.store().StoreProject(p, true)
	return p, storeErr(err)
}
