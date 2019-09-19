package sort

import (
	"errors"
	"fmt"
	"strings"

	"git.sr.ht/~sircmpwn/aerc/worker/types"
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
	default:
		return types.SortArrival, fmt.Errorf("%v is not a valid sort criterion", arg)
	}
}
