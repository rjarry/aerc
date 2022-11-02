package types

import (
	"fmt"
	"strings"
	"testing"
)

func genFakeTree() *Thread {
	tree := &Thread{
		Uid: 0,
	}
	var prevChild *Thread
	for i := 1; i < 3; i++ {
		child := &Thread{
			Uid:         uint32(i * 10),
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
		for j := 1; j < 3; j++ {
			second := &Thread{
				Uid:         child.Uid + uint32(j),
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
			limit := 3
			if j == 2 {
				limit = 8
			}
			for k := 1; k < limit; k++ {
				third := &Thread{
					Uid:         second.Uid*10 + uint32(k),
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
		seq = append(seq, fmt.Sprintf("%d", t.Uid))
		return nil
	})
	return strings.Join(seq, ".")
}

func TestThread_AddChild(t *testing.T) {
	tests := []struct {
		name string
		seq  []int
		want string
	}{
		{
			name: "ascending",
			seq:  []int{1, 2, 3, 4, 5, 6},
			want: "0.1.2.3.4.5.6",
		},
		{
			name: "descending",
			seq:  []int{6, 5, 4, 3, 2, 1},
			want: "0.6.5.4.3.2.1",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tree := &Thread{Uid: 0}
			for _, i := range test.seq {
				tree.AddChild(&Thread{Uid: uint32(i)})
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
		seq  []int
		want string
	}{
		{
			name: "ascending",
			seq:  []int{1, 2, 3, 4, 5, 6},
			want: "0.1.2.3.4.5.6",
		},
		{
			name: "descending",
			seq:  []int{6, 5, 4, 3, 2, 1},
			want: "0.1.2.3.4.5.6",
		},
		{
			name: "mixed",
			seq:  []int{2, 1, 6, 3, 4, 5},
			want: "0.1.2.3.4.5.6",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tree := &Thread{Uid: 0}
			for _, i := range test.seq {
				tree.OrderedInsert(&Thread{Uid: uint32(i)})
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
		seq  []int
		want string
	}{
		{
			name: "ascending",
			seq:  []int{1, 2, 3, 4, 5, 6},
			want: "0.6.4.2.1.3.5",
		},
		{
			name: "descending",
			seq:  []int{6, 5, 4, 3, 2, 1},
			want: "0.6.4.2.1.3.5",
		},
		{
			name: "mixed",
			seq:  []int{2, 1, 6, 3, 4, 5},
			want: "0.6.4.2.1.3.5",
		},
	}
	sortMap := map[uint32]int{
		uint32(6): 1,
		uint32(4): 2,
		uint32(2): 3,
		uint32(1): 4,
		uint32(3): 5,
		uint32(5): 6,
	}

	// bigger compares the new child with the next node and returns true if
	// the child node is bigger and false otherwise.
	bigger := func(newNode, nextChild *Thread) bool {
		return sortMap[newNode.Uid] > sortMap[nextChild.Uid]
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tree := &Thread{Uid: 0}
			for _, i := range test.seq {
				tree.InsertCmp(&Thread{Uid: uint32(i)}, bigger)
			}
			if got := uidSeq(tree); got != test.want {
				t.Errorf("got: %s, but wanted: %s", got,
					test.want)
			}
		})
	}
}
