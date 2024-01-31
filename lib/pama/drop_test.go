package pama_test

import (
	"reflect"
	"testing"

	"git.sr.ht/~rjarry/aerc/lib/pama"
	"git.sr.ht/~rjarry/aerc/lib/pama/models"
)

func TestPatchmgmt_Drop(t *testing.T) {
	setup := func(p models.Project) (pama.PatchManager, models.RevisionController, models.PersistentStorer) {
		return newTestManager(
			[]string{"0", "1", "2", "3", "4", "5"},
			[]string{"0", "a", "b", "c", "d", "f"},
			map[string]models.Project{p.Name: p}, p.Name,
		)
	}

	tests := []struct {
		name    string
		drop    string
		commits []models.Commit
		want    []models.Commit
	}{
		{
			name: "drop only patch",
			drop: "patch1",
			commits: []models.Commit{
				newCommit("1", "a", "patch1"),
			},
			want: []models.Commit{},
		},
		{
			name: "drop second one of two patch",
			drop: "patch2",
			commits: []models.Commit{
				newCommit("1", "a", "patch1"),
				newCommit("2", "b", "patch2"),
			},
			want: []models.Commit{
				newCommit("1", "a", "patch1"),
			},
		},
		{
			name: "drop first one of two patch",
			drop: "patch1",
			commits: []models.Commit{
				newCommit("1", "a", "patch1"),
				newCommit("2", "b", "patch2"),
			},
			want: []models.Commit{
				newCommit("2_new", "b", "patch2"),
			},
		},
	}

	for _, test := range tests {
		p := models.Project{
			Name:    "project1",
			Commits: test.commits,
			Base:    newCommit("0", "0", ""),
		}
		mgr, rc, _ := setup(p)

		err := mgr.DropPatch(test.drop)
		if err != nil {
			t.Errorf("test '%s' failed. %v", test.name, err)
		}

		q, _ := mgr.CurrentProject()
		if !reflect.DeepEqual(q.Commits, test.want) {
			t.Errorf("test '%s' failed. Commits don't match: "+
				"got %v, but wanted %v", test.name, q.Commits,
				test.want)
		}

		if len(test.want) > 0 {
			last := test.want[len(test.want)-1]
			if !rc.Exists(last.ID) {
				t.Errorf("test '%s' failed. Could not find last commits: %v", test.name, last)
			}
		}
	}
}
