package pama

import (
	"errors"
	"io"
	"strings"

	"git.sr.ht/~rjarry/aerc/lib/log"
	"git.sr.ht/~rjarry/aerc/lib/pama/models"
)

func (m PatchManager) Projects(name string) ([]models.Project, error) {
	all, err := m.store().Projects()
	if err != nil {
		return nil, storeErr(err)
	}
	if len(name) == 0 {
		return all, nil
	}
	var projects []models.Project
	for _, p := range all {
		if strings.Contains(p.Name, name) {
			projects = append(projects, p)
		}
	}
	if len(projects) == 0 {
		return nil, errors.New("No projects found.")
	}
	return projects, nil
}

func (m PatchManager) NewReader(projects []models.Project) io.Reader {
	cur, err := m.CurrentProject()
	currentName := cur.Name
	if err != nil {
		log.Warnf("could not get current project: %v", err)
		currentName = ""
	}

	readers := make([]io.Reader, 0, len(projects))
	for _, p := range projects {
		rc, err := m.rc(p.RevctrlID, p.Root)
		if err != nil {
			log.Errorf("project '%s' failed with: %v", p.Name, err)
			continue
		}

		notes := make(map[string]string)
		for _, c := range p.Commits {
			if !rc.Exists(c.ID) {
				notes[c.ID] = "Rebase needed"
			}
		}

		active := p.Name == currentName && len(projects) > 1
		readers = append(readers, p.NewReader(active, notes))
	}
	return io.MultiReader(readers...)
}
