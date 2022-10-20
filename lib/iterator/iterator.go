package iterator

// Factory is the interface that wraps the NewIterator method. The
// NewIterator() creates either UID or thread iterators and ensures that both
// types of iterators implement the same iteration direction.
type Factory interface {
	NewIterator(a interface{}) Iterator
}

// Iterator implements an interface for iterating over UID or thread data. If
// Next() returns true, the current value of the iterator can be read with
// Value(). The return value of Value() is an interface{} type which needs to
// be cast to the correct type.
//
// The iterators are implemented such that the first returned value always
// represents the top message in the message list. Hence, StartIndex() would
// return the index of the top message whereas EndIndex() returns the index of
// message at the bottom of the list.
type Iterator interface {
	Next() bool
	Value() interface{}
	StartIndex() int
	EndIndex() int
}

// NewFactory creates an iterator factory. When reverse is true, the iterators
// are reversed in the sense that the lowest UID messages are displayed at the
// top of the message list. Otherwise, the default order is with the highest
// UID message on top.
func NewFactory(reverse bool) Factory {
	if reverse {
		return &reverseFactory{}
	}
	return &defaultFactory{}
}
