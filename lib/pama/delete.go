package pama

import "fmt"

// Delete removes provided project
func (m PatchManager) Delete(name string) error {
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

	cur, err := m.CurrentProject()
	if err == nil && cur.Name == name {
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

	return storeErr(m.store().DeleteProject(name))
}

func (m PatchManager) Names() ([]string, error) {
	names, err := m.store().Names()
	return names, storeErr(err)
}
