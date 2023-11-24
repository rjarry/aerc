package patch

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"git.sr.ht/~rjarry/aerc/lib/pama/models"
)

func TestRebase_reorder(t *testing.T) {
	newCommits := func(order []string) []models.Commit {
		var commits []models.Commit
		for _, s := range order {
			commits = append(commits, models.Commit{ID: s})
		}
		return commits
	}
	tests := []struct {
		name    string
		commits []models.Commit
		now     []string
		by      []string
		want    []models.Commit
	}{
		{
			name:    "nothing to reorder",
			commits: newCommits([]string{"1", "2", "3"}),
			now:     []string{"1", "2", "3"},
			by:      []string{"1", "2", "3"},
			want:    newCommits([]string{"1", "2", "3"}),
		},
		{
			name:    "reorder",
			commits: newCommits([]string{"1", "3", "2"}),
			now:     []string{"1", "3", "2"},
			by:      []string{"1", "2", "3"},
			want:    newCommits([]string{"1", "2", "3"}),
		},
		{
			name:    "reorder inverted",
			commits: newCommits([]string{"3", "2", "1"}),
			now:     []string{"3", "2", "1"},
			by:      []string{"1", "2", "3"},
			want:    newCommits([]string{"1", "2", "3"}),
		},
		{
			name:    "changed hash: do not sort",
			commits: newCommits([]string{"1", "6", "3"}),
			now:     []string{"1", "6", "3"},
			by:      []string{"1", "2", "3"},
			want:    newCommits([]string{"1", "6", "3"}),
		},
	}

	for _, test := range tests {
		reorder(test.commits, test.now, test.by)
		if !reflect.DeepEqual(test.commits, test.want) {
			t.Errorf("test '%s' failed to reorder: got %v but "+
				"want %v", test.name, test.commits, test.want)
		}
	}
}

func newCommit(id, subj, tag string) models.Commit {
	return models.Commit{
		ID:      id,
		Subject: subj,
		Tag:     tag,
	}
}

func TestRebase_parse(t *testing.T) {
	input := `
	# some header info
	hello_v1   123  same info
	hello_v1   456  same info
	untracked  789  same info
	hello_v2   012  diff info
	untracked  345  diff info # not very useful comment
	# some footer info
	`
	commits := []models.Commit{
		newCommit("123123", "same info", "hello_v1"),
		newCommit("456456", "same info", "hello_v1"),
		newCommit("789789", "same info", models.Untracked),
		newCommit("012012", "diff info", "hello_v2"),
		newCommit("345345", "diff info", models.Untracked),
	}

	var order []string
	for _, c := range commits {
		order = append(order, fmt.Sprintf("%3.3s", c.ID))
	}

	table := make(map[string]models.Commit)
	for i, shortId := range order {
		table[shortId] = commits[i]
	}

	rebase := &rebase{
		commits: commits,
		table:   table,
		order:   order,
	}

	results := rebase.parse(strings.NewReader(input))

	if len(results) != 3 {
		t.Errorf("failed to return correct number of commits: "+
			"got %d but wanted 3", len(results))
	}
}
