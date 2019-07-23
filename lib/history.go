package lib

// History represents a list of elements ordered by time.
type History interface {
	// Add a new element to the history
	Add(string)
	// Get the next element in history
	Next() string
	// Get the previous element in history
	Prev() string
	// Reset the current location in history
	Reset()
}
