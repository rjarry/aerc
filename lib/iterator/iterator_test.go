package iterator_test

import (
	"testing"

	"git.sr.ht/~rjarry/aerc/lib/iterator"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

func toThreads(uids []uint32) []*types.Thread {
	threads := make([]*types.Thread, len(uids))
	for i, u := range uids {
		threads[i] = &types.Thread{Uid: u}
	}
	return threads
}

func TestIterator_DefaultFactory(t *testing.T) {
	input := []uint32{1, 2, 3, 4, 5, 6, 7, 8, 9}
	want := []uint32{9, 8, 7, 6, 5, 4, 3, 2, 1}

	factory := iterator.NewFactory(false)
	if factory == nil {
		t.Errorf("could not create factory")
	}
	start, end := len(input)-1, 0
	checkUids(t, factory, input, want, start, end)
	checkThreads(t, factory, toThreads(input),
		toThreads(want), start, end)
}

func TestIterator_ReverseFactory(t *testing.T) {
	input := []uint32{1, 2, 3, 4, 5, 6, 7, 8, 9}
	want := []uint32{1, 2, 3, 4, 5, 6, 7, 8, 9}

	factory := iterator.NewFactory(true)
	if factory == nil {
		t.Errorf("could not create factory")
	}

	start, end := 0, len(input)-1
	checkUids(t, factory, input, want, start, end)
	checkThreads(t, factory, toThreads(input),
		toThreads(want), start, end)
}

func checkUids(t *testing.T, factory iterator.Factory,
	input []uint32, want []uint32, start, end int,
) {
	label := "uids"
	got := make([]uint32, 0)
	iter := factory.NewIterator(input)
	for iter.Next() {
		got = append(got, iter.Value().(uint32))
	}
	if len(got) != len(want) {
		t.Errorf(label + "number of elements not correct")
	}
	for i, u := range want {
		if got[i] != u {
			t.Errorf(label + "order not correct")
		}
	}
	if iter.StartIndex() != start {
		t.Errorf(label + "start index not correct")
	}
	if iter.EndIndex() != end {
		t.Errorf(label + "end index not correct")
	}
}

func checkThreads(t *testing.T, factory iterator.Factory,
	input []*types.Thread, want []*types.Thread, start, end int,
) {
	label := "threads"
	got := make([]*types.Thread, 0)
	iter := factory.NewIterator(input)
	for iter.Next() {
		got = append(got, iter.Value().(*types.Thread))
	}
	if len(got) != len(want) {
		t.Errorf(label + "number of elements not correct")
	}
	for i, th := range want {
		if got[i].Uid != th.Uid {
			t.Errorf(label + "order not correct")
		}
	}
	if iter.StartIndex() != start {
		t.Errorf(label + "start index not correct")
	}
	if iter.EndIndex() != end {
		t.Errorf(label + "end index not correct")
	}
}
