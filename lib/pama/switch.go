package pama

import (
	"fmt"
	"regexp"
	"time"

	"git.sr.ht/~rjarry/aerc/lib/log"
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

var switchDebouncer *time.Timer

func DebouncedSwitchProject(name string) {
	if switchDebouncer != nil {
		if switchDebouncer.Stop() {
			log.Debugf("pama: switch debounced")
		}
	}
	if name == "" {
		return
	}
	switchDebouncer = time.AfterFunc(500*time.Millisecond, func() {
		if err := New().SwitchProject(name); err != nil {
			log.Debugf("could not switch to project %s: %v",
				name, err)
		} else {
			log.Debugf("project switch to project %s", name)
		}
	})
}

var fromSubject = regexp.MustCompile(
	`\[\s*(RFC|DRAFT|[Dd]raft)*\s*(PATCH|[Pp]atch)\s+([^\s\]]+)\s*[vV]*[0-9/]*\s*\] `)

func FromSubject(s string) string {
	matches := fromSubject.FindStringSubmatch(s)
	if len(matches) >= 3 {
		return matches[3]
	}
	return ""
}
