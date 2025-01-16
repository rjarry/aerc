package iterator_test

import (
	"testing"

	"git.sr.ht/~rjarry/aerc/lib/iterator"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

func toThreads(uids []models.UID) []*types.Thread {
	threads := make([]*types.Thread, len(uids))
	for i, u := range uids {
		threads[i] = &types.Thread{Uid: u}
	}
	return threads
}

func TestIterator_DefaultFactory(t *testing.T) {
	input := []models.UID{"1", "2", "3", "4", "5", "6", "7", "8", "9"}
	want := []models.UID{"9", "8", "7", "6", "5", "4", "3", "2", "1"}

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
	input := []models.UID{"1", "2", "3", "4", "5", "6", "7", "8", "9"}
	want := []models.UID{"1", "2", "3", "4", "5", "6", "7", "8", "9"}

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
	input []models.UID, want []models.UID, start, end int,
) {
	label := "uids"
	got := make([]models.UID, 0)
	iter := factory.NewIterator(input)
	for iter.Next() {
		got = append(got, iter.Value().(models.UID))
	}
	if len(got) != len(want) {
		t.Errorf("%s: number of elements not correct", label)
	}
	for i, u := range want {
		if got[i] != u {
			t.Errorf("%s: order not correct", label)
		}
	}
	if iter.StartIndex() != start {
		t.Errorf("%s: start index not correct", label)
	}
	if iter.EndIndex() != end {
		t.Errorf("%s: end index not correct", label)
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
		t.Errorf("%s: number of elements not correct", label)
	}
	for i, th := range want {
		if got[i].Uid != th.Uid {
			t.Errorf("%s: order not correct", label)
		}
	}
	if iter.StartIndex() != start {
		t.Errorf("%s: start index not correct", label)
	}
	if iter.EndIndex() != end {
		t.Errorf("%s: end index not correct", label)
	}
}
