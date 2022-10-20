package iterator_test

import (
	"testing"

	"git.sr.ht/~rjarry/aerc/lib/iterator"
)

type indexer struct {
	start int
	end   int
}

func (ip *indexer) StartIndex() int {
	return ip.start
}

func (ip *indexer) EndIndex() int {
	return ip.end
}

func TestMoveIndex(t *testing.T) {
	tests := []struct {
		idx      int
		delta    int
		start    int
		end      int
		cb       iterator.BoundsCheckFunc
		expected int
	}{
		{
			idx:      0,
			delta:    1,
			start:    0,
			end:      2,
			cb:       iterator.FixBounds,
			expected: 1,
		},
		{
			idx:      0,
			delta:    5,
			start:    0,
			end:      2,
			cb:       iterator.FixBounds,
			expected: 2,
		},
		{
			idx:      0,
			delta:    -1,
			start:    0,
			end:      2,
			cb:       iterator.FixBounds,
			expected: 0,
		},
		{
			idx:      0,
			delta:    2,
			start:    0,
			end:      2,
			cb:       iterator.WrapBounds,
			expected: 2,
		},
		{
			idx:      0,
			delta:    3,
			start:    0,
			end:      2,
			cb:       iterator.WrapBounds,
			expected: 0,
		},
		{
			idx:      0,
			delta:    -1,
			start:    0,
			end:      2,
			cb:       iterator.WrapBounds,
			expected: 2,
		},
		{
			idx:      2,
			delta:    2,
			start:    0,
			end:      2,
			cb:       iterator.WrapBounds,
			expected: 1,
		},
		{
			idx:      0,
			delta:    -2,
			start:    0,
			end:      2,
			cb:       iterator.WrapBounds,
			expected: 1,
		},
		{
			idx:      1,
			delta:    1,
			start:    2,
			end:      0,
			cb:       iterator.FixBounds,
			expected: 0,
		},
		{
			idx:      0,
			delta:    1,
			start:    2,
			end:      0,
			cb:       iterator.FixBounds,
			expected: 0,
		},
		{
			idx:      0,
			delta:    1,
			start:    2,
			end:      0,
			cb:       iterator.WrapBounds,
			expected: 2,
		},
	}

	for i, test := range tests {
		idx := iterator.MoveIndex(
			test.idx,
			test.delta,
			&indexer{test.start, test.end},
			test.cb,
		)
		if idx != test.expected {
			t.Errorf("test %d [%#v] failed: got %d but expected %d",
				i, test, idx, test.expected)
		}
	}
}
