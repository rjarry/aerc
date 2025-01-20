package pama_test

import (
	"errors"

	"git.sr.ht/~rjarry/aerc/lib/pama"
	"git.sr.ht/~rjarry/aerc/lib/pama/models"
)

var errNotFound = errors.New("not found")

func newCommit(id, subj, tag string) models.Commit {
	return models.Commit{ID: id, Subject: subj, Tag: tag}
}

func newTestManager(
	commits []string,
	subjects []string,
	data map[string]models.Project,
	current string,
) (pama.PatchManager, models.RevisionController, models.PersistentStorer) {
	rc := mockRevctrl{
		commitIDs: commits,
		titles:    subjects,
	}
	store := mockStore{
		data:    data,
		current: current,
	}
	return pama.FromFunc(
		nil,
		func(_ string, _ string) (models.RevisionController, error) {
			return &rc, nil
		},
		func() models.PersistentStorer {
			return &store
		},
	), &rc, &store
}

type mockRevctrl struct {
	commitIDs []string
	titles    []string
}

func (c *mockRevctrl) Support() bool {
	return true
}

func (c *mockRevctrl) Clean() bool {
	return true
}

func (c *mockRevctrl) Root() (string, error) {
	return "", nil
}

func (c *mockRevctrl) Head() (string, error) {
	return c.commitIDs[len(c.commitIDs)-1], nil
}

func (c *mockRevctrl) History(commit string) ([]string, error) {
	for i, s := range c.commitIDs {
		if s == commit {
			cp := make([]string, len(c.commitIDs[i+1:]))
			copy(cp, c.commitIDs[i+1:])
			return cp, nil
		}
	}
	return nil, errNotFound
}

func (c *mockRevctrl) Exists(commit string) bool {
	for _, s := range c.commitIDs {
		if s == commit {
			return true
		}
	}
	return false
}

func (c *mockRevctrl) Subject(commit string) string {
	for i, s := range c.commitIDs {
		if s == commit {
			return c.titles[i]
		}
	}
	return ""
}

func (c *mockRevctrl) Author(commit string) string {
	return ""
}

func (c *mockRevctrl) Date(commit string) string {
	return ""
}

func (c *mockRevctrl) Drop(commit string) error {
	for i, s := range c.commitIDs {
		if s == commit {
			c.commitIDs = append(c.commitIDs[:i], c.commitIDs[i+1:]...)
			c.titles = append(c.titles[:i], c.titles[i+1:]...)
			// modify commitIDs to simulate a "real" change in
			// commit history that will also change all subsequent
			// commitIDs
			for j := i; j < len(c.commitIDs); j++ {
				c.commitIDs[j] += "_new"
			}
			return nil
		}
	}
	return errNotFound
}

func (c *mockRevctrl) CreateWorktree(_, _ string) error {
	return nil
}

func (c *mockRevctrl) DeleteWorktree(_ string) error {
	return nil
}

func (c *mockRevctrl) ApplyCmd() string {
	return ""
}

type mockStore struct {
	data    map[string]models.Project
	current string
}

func (s *mockStore) StoreProject(p models.Project, ow bool) error {
	_, ok := s.data[p.Name]
	if ok && !ow {
		return errors.New("already there")
	}
	s.data[p.Name] = p
	return nil
}

func (s *mockStore) DeleteProject(name string) error {
	delete(s.data, name)
	return nil
}

func (s *mockStore) CurrentName() (string, error) {
	return s.current, nil
}

func (s *mockStore) SetCurrent(c string) error {
	s.current = c
	return nil
}

func (s *mockStore) Current() (models.Project, error) {
	return s.data[s.current], nil
}

func (s *mockStore) Names() ([]string, error) {
	var names []string
	for name := range s.data {
		names = append(names, name)
	}
	return names, nil
}

func (s *mockStore) Project(_ string) (models.Project, error) {
	return models.Project{}, nil
}

func (s *mockStore) Projects() ([]models.Project, error) {
	var ps []models.Project
	for _, p := range s.data {
		ps = append(ps, p)
	}
	return ps, nil
}
