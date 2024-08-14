package types

import (
	"fmt"
	"strings"
	"testing"

	"git.sr.ht/~rjarry/aerc/models"
)

func genFakeTree() *Thread {
	tree := new(Thread)
	var prevChild *Thread
	for i := uint32(1); i < uint32(3); i++ {
		child := &Thread{
			Uid:         models.Uint32ToUid(i * 10),
			Parent:      tree,
			PrevSibling: prevChild,
		}
		if prevChild != nil {
			prevChild.NextSibling = child
		} else if tree.FirstChild == nil {
			tree.FirstChild = child
		} else {
			panic("unreachable")
		}
		prevChild = child
		var prevSecond *Thread
		for j := uint32(1); j < uint32(3); j++ {
			second := &Thread{
				Uid:         models.Uint32ToUid(models.UidToUint32(child.Uid) + j),
				Parent:      child,
				PrevSibling: prevSecond,
			}
			if prevSecond != nil {
				prevSecond.NextSibling = second
			} else if child.FirstChild == nil {
				child.FirstChild = second
			} else {
				panic("unreachable")
			}
			prevSecond = second
			var prevThird *Thread
			limit := uint32(3)
			if j == 2 {
				limit = 8
			}
			for k := uint32(1); k < limit; k++ {
				third := &Thread{
					Uid:         models.Uint32ToUid(models.UidToUint32(second.Uid)*10 + j),
					Parent:      second,
					PrevSibling: prevThird,
				}
				if prevThird != nil {
					prevThird.NextSibling = third
				} else if second.FirstChild == nil {
					second.FirstChild = third
				} else {
					panic("unreachable")
				}
				prevThird = third
			}
		}
	}
	return tree
}

func TestNewWalk(t *testing.T) {
	tree := genFakeTree()
	var prefix []string
	lastLevel := 0
	tree.Walk(func(t *Thread, lvl int, e error) error {
		if e != nil {
			fmt.Printf("ERROR: %v\n", e)
		}
		if lvl > lastLevel && lvl > 1 {
			// we actually just descended... so figure out what connector we need
			// level 1 is flush to the root, so we avoid the indentation there
			if t.Parent.NextSibling != nil {
				prefix = append(prefix, "│  ")
			} else {
				prefix = append(prefix, "   ")
			}
		} else if lvl < lastLevel {
			// ascended, need to trim the prefix layers
			diff := lastLevel - lvl
			prefix = prefix[:len(prefix)-diff]
		}

		var arrow string
		if t.Parent != nil {
			if t.NextSibling != nil {
				arrow = "├─>"
			} else {
				arrow = "└─>"
			}
		}

		// format
		fmt.Printf("%s%s%s\n", strings.Join(prefix, ""), arrow, t)

		lastLevel = lvl
		return nil
	})
}

func uidSeq(tree *Thread) string {
	var seq []string
	tree.Walk(func(t *Thread, _ int, _ error) error {
		seq = append(seq, string(t.Uid))
		return nil
	})
	return strings.Join(seq, ".")
}

func TestThread_AddChild(t *testing.T) {
	tests := []struct {
		name string
		seq  []models.UID
		want string
	}{
		{
			name: "ascending",
			seq:  []models.UID{"1", "2", "3", "4", "5", "6"},
			want: ".1.2.3.4.5.6",
		},
		{
			name: "descending",
			seq:  []models.UID{"6", "5", "4", "3", "2", "1"},
			want: ".6.5.4.3.2.1",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tree := new(Thread)
			for _, i := range test.seq {
				tree.AddChild(&Thread{Uid: i})
			}
			if got := uidSeq(tree); got != test.want {
				t.Errorf("got: %s, but wanted: %s", got,
					test.want)
			}
		})
	}
}

func TestThread_OrderedInsert(t *testing.T) {
	tests := []struct {
		name string
		seq  []models.UID
		want string
	}{
		{
			name: "ascending",
			seq:  []models.UID{"1", "2", "3", "4", "5", "6"},
			want: ".1.2.3.4.5.6",
		},
		{
			name: "descending",
			seq:  []models.UID{"6", "5", "4", "3", "2", "1"},
			want: ".1.2.3.4.5.6",
		},
		{
			name: "mixed",
			seq:  []models.UID{"2", "1", "6", "3", "4", "5"},
			want: ".1.2.3.4.5.6",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tree := new(Thread)
			for _, i := range test.seq {
				tree.OrderedInsert(&Thread{Uid: i})
			}
			if got := uidSeq(tree); got != test.want {
				t.Errorf("got: %s, but wanted: %s", got,
					test.want)
			}
		})
	}
}

func TestThread_InsertCmd(t *testing.T) {
	tests := []struct {
		name string
		seq  []models.UID
		want string
	}{
		{
			name: "ascending",
			seq:  []models.UID{"1", "2", "3", "4", "5", "6"},
			want: ".6.4.2.1.3.5",
		},
		{
			name: "descending",
			seq:  []models.UID{"6", "5", "4", "3", "2", "1"},
			want: ".6.4.2.1.3.5",
		},
		{
			name: "mixed",
			seq:  []models.UID{"2", "1", "6", "3", "4", "5"},
			want: ".6.4.2.1.3.5",
		},
	}
	sortMap := map[models.UID]int{
		"6": 1,
		"4": 2,
		"2": 3,
		"1": 4,
		"3": 5,
		"5": 6,
	}

	// bigger compares the new child with the next node and returns true if
	// the child node is bigger and false otherwise.
	bigger := func(newNode, nextChild *Thread) bool {
		return sortMap[newNode.Uid] > sortMap[nextChild.Uid]
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tree := new(Thread)
			for _, i := range test.seq {
				tree.InsertCmp(&Thread{Uid: i}, bigger)
			}
			if got := uidSeq(tree); got != test.want {
				t.Errorf("got: %s, but wanted: %s", got,
					test.want)
			}
		})
	}
}
