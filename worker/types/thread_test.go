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
