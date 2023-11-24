package pama

import (
	"fmt"
)

func (m PatchManager) SwitchProject(name string) error {
	c, err := m.CurrentProject()
	if err == nil {
		if c.Name == name {
			return nil
		}
	}
	names, err := m.store().Names()
	if err != nil {
		return storeErr(err)
	}
	found := false
	for _, n := range names {
		if n == name {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("Project '%s' not found", name)
	}
	return storeErr(m.store().SetCurrent(name))
}
