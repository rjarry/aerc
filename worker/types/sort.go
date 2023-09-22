package types

type SortField int

const (
	SortArrival SortField = iota
	SortCc
	SortDate
	SortFrom
	SortRead
	SortSize
	SortSubject
	SortTo
	SortFlagged
)

type SortCriterion struct {
	Field   SortField
	Reverse bool
}
