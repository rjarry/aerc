package lib

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"git.sr.ht/~sircmpwn/aerc/models"
	"git.sr.ht/~sircmpwn/aerc/worker/types"
)

func Sort(messageInfos []*models.MessageInfo,
	criteria []*types.SortCriterion) ([]uint32, error) {
	// loop through in reverse to ensure we sort by non-primary fields first
	for i := len(criteria) - 1; i >= 0; i-- {
		criterion := criteria[i]
		var err error
		switch criterion.Field {
		case types.SortArrival:
			err = sortDate(messageInfos, criterion,
				func(msgInfo *models.MessageInfo) time.Time {
					return msgInfo.InternalDate
				})
		case types.SortCc:
			err = sortAddresses(messageInfos, criterion,
				func(msgInfo *models.MessageInfo) []*models.Address {
					return msgInfo.Envelope.Cc
				})
		case types.SortDate:
			err = sortDate(messageInfos, criterion,
				func(msgInfo *models.MessageInfo) time.Time {
					return msgInfo.Envelope.Date
				})
		case types.SortFrom:
			err = sortAddresses(messageInfos, criterion,
				func(msgInfo *models.MessageInfo) []*models.Address {
					return msgInfo.Envelope.From
				})
		case types.SortRead:
			err = sortFlags(messageInfos, criterion, models.SeenFlag)
		case types.SortSize:
			err = sortInts(messageInfos, criterion,
				func(msgInfo *models.MessageInfo) uint32 {
					return msgInfo.Size
				})
		case types.SortSubject:
			err = sortStrings(messageInfos, criterion,
				func(msgInfo *models.MessageInfo) string {
					subject := strings.ToLower(msgInfo.Envelope.Subject)
					subject = strings.TrimPrefix(subject, "re: ")
					return strings.TrimPrefix(subject, "fwd: ")
				})
		case types.SortTo:
			err = sortAddresses(messageInfos, criterion,
				func(msgInfo *models.MessageInfo) []*models.Address {
					return msgInfo.Envelope.To
				})
		}
		if err != nil {
			return nil, err
		}
	}
	var uids []uint32
	// copy in reverse as msgList displays backwards
	for i := len(messageInfos) - 1; i >= 0; i-- {
		uids = append(uids, messageInfos[i].Uid)
	}
	return uids, nil
}

func sortDate(messageInfos []*models.MessageInfo, criterion *types.SortCriterion,
	getValue func(*models.MessageInfo) time.Time) error {
	var slice []*dateStore
	for _, msgInfo := range messageInfos {
		slice = append(slice, &dateStore{
			Value:   getValue(msgInfo),
			MsgInfo: msgInfo,
		})
	}
	sortSlice(criterion, dateSlice{slice})
	for i := 0; i < len(messageInfos); i++ {
		messageInfos[i] = slice[i].MsgInfo
	}
	return nil
}

func sortAddresses(messageInfos []*models.MessageInfo, criterion *types.SortCriterion,
	getValue func(*models.MessageInfo) []*models.Address) error {
	var slice []*addressStore
	for _, msgInfo := range messageInfos {
		slice = append(slice, &addressStore{
			Value:   getValue(msgInfo),
			MsgInfo: msgInfo,
		})
	}
	sortSlice(criterion, addressSlice{slice})
	for i := 0; i < len(messageInfos); i++ {
		messageInfos[i] = slice[i].MsgInfo
	}
	return nil
}

func sortFlags(messageInfos []*models.MessageInfo, criterion *types.SortCriterion,
	testFlag models.Flag) error {
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
	sortSlice(criterion, boolSlice{slice})
	for i := 0; i < len(messageInfos); i++ {
		messageInfos[i] = slice[i].MsgInfo
	}
	return nil
}

func sortInts(messageInfos []*models.MessageInfo, criterion *types.SortCriterion,
	getValue func(*models.MessageInfo) uint32) error {
	var slice []*intStore
	for _, msgInfo := range messageInfos {
		slice = append(slice, &intStore{
			Value:   getValue(msgInfo),
			MsgInfo: msgInfo,
		})
	}
	sortSlice(criterion, intSlice{slice})
	for i := 0; i < len(messageInfos); i++ {
		messageInfos[i] = slice[i].MsgInfo
	}
	return nil
}

func sortStrings(messageInfos []*models.MessageInfo, criterion *types.SortCriterion,
	getValue func(*models.MessageInfo) string) error {
	var slice []*lexiStore
	for _, msgInfo := range messageInfos {
		slice = append(slice, &lexiStore{
			Value:   getValue(msgInfo),
			MsgInfo: msgInfo,
		})
	}
	sortSlice(criterion, lexiSlice{slice})
	for i := 0; i < len(messageInfos); i++ {
		messageInfos[i] = slice[i].MsgInfo
	}
	return nil
}

type lexiStore struct {
	Value   string
	MsgInfo *models.MessageInfo
}

type lexiSlice struct{ Slice []*lexiStore }

func (s lexiSlice) Len() int      { return len(s.Slice) }
func (s lexiSlice) Swap(i, j int) { s.Slice[i], s.Slice[j] = s.Slice[j], s.Slice[i] }
func (s lexiSlice) Less(i, j int) bool {
	return s.Slice[i].Value < s.Slice[j].Value
}

type dateStore struct {
	Value   time.Time
	MsgInfo *models.MessageInfo
}

type dateSlice struct{ Slice []*dateStore }

func (s dateSlice) Len() int      { return len(s.Slice) }
func (s dateSlice) Swap(i, j int) { s.Slice[i], s.Slice[j] = s.Slice[j], s.Slice[i] }
func (s dateSlice) Less(i, j int) bool {
	return s.Slice[i].Value.Before(s.Slice[j].Value)
}

type intStore struct {
	Value   uint32
	MsgInfo *models.MessageInfo
}

type intSlice struct{ Slice []*intStore }

func (s intSlice) Len() int      { return len(s.Slice) }
func (s intSlice) Swap(i, j int) { s.Slice[i], s.Slice[j] = s.Slice[j], s.Slice[i] }
func (s intSlice) Less(i, j int) bool {
	return s.Slice[i].Value < s.Slice[j].Value
}

type addressStore struct {
	Value   []*models.Address
	MsgInfo *models.MessageInfo
}

type addressSlice struct{ Slice []*addressStore }

func (s addressSlice) Len() int      { return len(s.Slice) }
func (s addressSlice) Swap(i, j int) { s.Slice[i], s.Slice[j] = s.Slice[j], s.Slice[i] }
func (s addressSlice) Less(i, j int) bool {
	addressI, addressJ := s.Slice[i].Value, s.Slice[j].Value
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
}

type boolStore struct {
	Value   bool
	MsgInfo *models.MessageInfo
}

type boolSlice struct{ Slice []*boolStore }

func (s boolSlice) Len() int      { return len(s.Slice) }
func (s boolSlice) Swap(i, j int) { s.Slice[i], s.Slice[j] = s.Slice[j], s.Slice[i] }
func (s boolSlice) Less(i, j int) bool {
	valI, valJ := s.Slice[i].Value, s.Slice[j].Value
	return valI && !valJ
}

func sortSlice(criterion *types.SortCriterion, interfce sort.Interface) {
	if criterion.Reverse {
		sort.Stable(sort.Reverse(interfce))
	} else {
		sort.Stable(interfce)
	}
}
