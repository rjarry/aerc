package sort

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"git.sr.ht/~rjarry/aerc/worker/types"
)

func GetSortCriteria(args []string) ([]*types.SortCriterion, error) {
	var sortCriteria []*types.SortCriterion
	reverse := false
	for _, arg := range args {
		if arg == "-r" {
			reverse = true
			continue
		}
		field, err := parseSortField(arg)
		if err != nil {
			return nil, err
		}
		sortCriteria = append(sortCriteria, &types.SortCriterion{
			Field:   field,
			Reverse: reverse,
		})
		reverse = false
	}
	if reverse {
		return nil, errors.New("Expected argument to reverse")
	}
	return sortCriteria, nil
}

func parseSortField(arg string) (types.SortField, error) {
	switch strings.ToLower(arg) {
	case "arrival":
		return types.SortArrival, nil
	case "cc":
		return types.SortCc, nil
	case "date":
		return types.SortDate, nil
	case "from":
		return types.SortFrom, nil
	case "read":
		return types.SortRead, nil
	case "size":
		return types.SortSize, nil
	case "subject":
		return types.SortSubject, nil
	case "to":
		return types.SortTo, nil
	case "flagged":
		return types.SortFlagged, nil
	default:
		return types.SortArrival, fmt.Errorf("%v is not a valid sort criterion", arg)
	}
}

// Sorts toSort by sortBy so that toSort becomes a permutation following the
// order of sortBy.
// toSort should be a subset of sortBy
func SortBy(toSort []uint32, sortBy []uint32) {
	// build a map from sortBy
	uidMap := make(map[uint32]int)
	for i, uid := range sortBy {
		uidMap[uid] = i
	}
	// sortslice of toSort with less function of indexing the map sortBy
	sort.Slice(toSort, func(i, j int) bool {
		return uidMap[toSort[i]] < uidMap[toSort[j]]
	})
}

// SortStringBy sorts the string slice s according to the order given in the
// order string slice.
func SortStringBy(s []string, order []string) {
	m := make(map[string]int)
	for i, d := range order {
		m[d] = i
	}
	sort.Slice(s, func(i, j int) bool {
		return m[s[i]] < m[s[j]]
	})
}
