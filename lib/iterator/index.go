package iterator

// IndexProvider implements a subset of the Interator interface
type IndexProvider interface {
	StartIndex() int
	EndIndex() int
}

// FixBounds will force the index i to either its lower- or upper-bound value
// if out-of-bound
func FixBounds(i, lower, upper int) int {
	switch {
	case i > upper:
		i = upper
	case i < lower:
		i = lower
	}
	return i
}

// WrapBounds will wrap the index i around its upper- or lower-bound if
// out-of-bound
func WrapBounds(i, lower, upper int) int {
	switch {
	case i > upper:
		i = lower + (i-upper-1)%upper
	case i < lower:
		i = upper - (lower-i-1)%upper
	}
	return i
}

type BoundsCheckFunc func(int, int, int) int

// MoveIndex moves the index variable idx forward by delta steps and ensures
// that the boundary policy as defined by the CheckBoundsFunc is enforced.
//
// If CheckBoundsFunc is nil, fix boundary checks are performed.
func MoveIndex(idx, delta int, indexer IndexProvider, cb BoundsCheckFunc) int {
	lower, upper := indexer.StartIndex(), indexer.EndIndex()
	sign := 1
	if upper < lower {
		lower, upper = upper, lower
		sign = -1
	}
	result := idx + sign*delta
	if cb == nil {
		return FixBounds(result, lower, upper)
	}
	return cb(result, lower, upper)
}
