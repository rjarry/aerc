package models

import (
	"fmt"
	"slices"
	"strings"
)

const (
	Untracked = "untracked"
)

func NewCommit(r RevisionController, id, tag string) Commit {
	return Commit{
		ID:        id,
		Subject:   r.Subject(id),
		Author:    r.Author(id),
		Date:      r.Date(id),
		MessageId: "",
		Tag:       tag,
	}
}

func (c Commit) Untracked() bool {
	return c.Tag == Untracked
}

func (c Commit) Info() string {
	s := []string{}
	if c.Subject == "" {
		s = append(s, "(no subject)")
	} else {
		s = append(s, c.Subject)
	}
	if c.Author != "" {
		s = append(s, c.Author)
	}
	if c.Date != "" {
		s = append(s, c.Date)
	}
	if c.MessageId != "" {
		s = append(s, "<"+c.MessageId+">")
	}
	return strings.Join(s, ", ")
}

func (c Commit) String() string {
	return fmt.Sprintf("%-6.6s %s", c.ID, c.Info())
}

type Commits []Commit

func (h Commits) Tags() []string {
	var tags []string
	dedup := make(map[string]struct{})
	for _, c := range h {
		_, ok := dedup[c.Tag]
		if ok {
			continue
		}
		tags = append(tags, c.Tag)
		dedup[c.Tag] = struct{}{}
	}
	return tags
}

func (h Commits) HasTag(t string) bool {
	for _, c := range h {
		if c.Tag == t {
			return true
		}
	}
	return false
}

func (h Commits) Lookup(id string) (Commit, bool) {
	for _, c := range h {
		if c.ID == id {
			return c, true
		}
	}
	return Commit{}, false
}

type CommitIDs []string

func (c CommitIDs) Has(id string) bool {
	return slices.Contains(c, id)
}
