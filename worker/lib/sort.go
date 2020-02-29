package lib

import (
	"fmt"
	"sort"
	"strings"

	"git.sr.ht/~sircmpwn/aerc/models"
	"git.sr.ht/~sircmpwn/aerc/worker/types"
)

func Sort(messageInfos []*models.MessageInfo,
	criteria []*types.SortCriterion) ([]uint32, error) {
	// loop through in reverse to ensure we sort by non-primary fields first
	for i := len(criteria) - 1; i >= 0; i-- {
		criterion := criteria[i]
		switch criterion.Field {
		case types.SortArrival:
			sortSlice(criterion, messageInfos, func(i, j int) bool {
				return messageInfos[i].InternalDate.Before(messageInfos[j].InternalDate)
			})
		case types.SortCc:
			sortAddresses(messageInfos, criterion,
				func(msgInfo *models.MessageInfo) []*models.Address {
					return msgInfo.Envelope.Cc
				})
		case types.SortDate:
			sortSlice(criterion, messageInfos, func(i, j int) bool {
				return messageInfos[i].Envelope.Date.Before(messageInfos[j].Envelope.Date)
			})
		case types.SortFrom:
			sortAddresses(messageInfos, criterion,
				func(msgInfo *models.MessageInfo) []*models.Address {
					return msgInfo.Envelope.From
				})
		case types.SortRead:
			sortFlags(messageInfos, criterion, models.SeenFlag)
		case types.SortSize:
			sortSlice(criterion, messageInfos, func(i, j int) bool {
				return messageInfos[i].Size < messageInfos[j].Size
			})
		case types.SortSubject:
			sortStrings(messageInfos, criterion,
				func(msgInfo *models.MessageInfo) string {
					subject := strings.ToLower(msgInfo.Envelope.Subject)
					subject = strings.TrimPrefix(subject, "re: ")
					return strings.TrimPrefix(subject, "fwd: ")
				})
		case types.SortTo:
			sortAddresses(messageInfos, criterion,
				func(msgInfo *models.MessageInfo) []*models.Address {
					return msgInfo.Envelope.To
				})
		}
	}
	var uids []uint32
	// copy in reverse as msgList displays backwards
	for i := len(messageInfos) - 1; i >= 0; i-- {
		uids = append(uids, messageInfos[i].Uid)
	}
	return uids, nil
}

func sortAddresses(messageInfos []*models.MessageInfo, criterion *types.SortCriterion,
	getValue func(*models.MessageInfo) []*models.Address) {
	sortSlice(criterion, messageInfos, func(i, j int) bool {
		addressI, addressJ := getValue(messageInfos[i]), getValue(messageInfos[j])
		var firstI, firstJ *models.Address
		if len(addressI) > 0 {
			firstI = addressI[0]
		}
		if len(addressJ) > 0 {
			firstJ = addressJ[0]
		}
		if firstI == nil && firstJ == nil {
			return false
		} else if firstI == nil && firstJ != nil {
			return false
		} else if firstI != nil && firstJ == nil {
			return true
		} else /* firstI != nil && firstJ != nil */ {
			getName := func(addr *models.Address) string {
				if addr.Name != "" {
					return addr.Name
				} else {
					return fmt.Sprintf("%s@%s", addr.Mailbox, addr.Host)
				}
			}
			return getName(firstI) < getName(firstJ)
		}
	})
}

func sortFlags(messageInfos []*models.MessageInfo, criterion *types.SortCriterion,
	testFlag models.Flag) {
	var slice []*boolStore
	for _, msgInfo := range messageInfos {
		flagPresent := false
		for _, flag := range msgInfo.Flags {
			if flag == testFlag {
				flagPresent = true
			}
		}
		slice = append(slice, &boolStore{
			Value:   flagPresent,
			MsgInfo: msgInfo,
		})
	}
	sortSlice(criterion, slice, func(i, j int) bool {
		valI, valJ := slice[i].Value, slice[j].Value
		return valI && !valJ
	})
	for i := 0; i < len(messageInfos); i++ {
		messageInfos[i] = slice[i].MsgInfo
	}
}

func sortStrings(messageInfos []*models.MessageInfo, criterion *types.SortCriterion,
	getValue func(*models.MessageInfo) string) {
	var slice []*lexiStore
	for _, msgInfo := range messageInfos {
		slice = append(slice, &lexiStore{
			Value:   getValue(msgInfo),
			MsgInfo: msgInfo,
		})
	}
	sortSlice(criterion, slice, func(i, j int) bool {
		return slice[i].Value < slice[j].Value
	})
	for i := 0; i < len(messageInfos); i++ {
		messageInfos[i] = slice[i].MsgInfo
	}
}

type lexiStore struct {
	Value   string
	MsgInfo *models.MessageInfo
}

type boolStore struct {
	Value   bool
	MsgInfo *models.MessageInfo
}

func sortSlice(criterion *types.SortCriterion, slice interface{}, less func(i, j int) bool) {
	if criterion.Reverse {
		sort.SliceStable(slice, func(i, j int) bool {
			return less(j, i)
		})
	} else {
		sort.SliceStable(slice, less)
	}
}
