package state

import (
	"fmt"
)

type AccountState struct {
	Connected    bool
	connActivity string
	passthrough  bool
	folders      map[string]*folderState
}

type folderState struct {
	Search         string
	Filter         string
	FilterActivity string
	Sorting        bool
	Threading      bool
}

func (s *AccountState) folderState(folder string) *folderState {
	if s.folders == nil {
		s.folders = make(map[string]*folderState)
	}
	if _, ok := s.folders[folder]; !ok {
		s.folders[folder] = &folderState{}
	}
	return s.folders[folder]
}

type SetStateFunc func(s *AccountState, folder string)

func SetConnected(state bool) SetStateFunc {
	return func(s *AccountState, folder string) {
		s.connActivity = ""
		s.Connected = state
	}
}

func ConnectionActivity(desc string) SetStateFunc {
	return func(s *AccountState, folder string) {
		s.connActivity = desc
	}
}

func SearchFilterClear() SetStateFunc {
	return func(s *AccountState, folder string) {
		s.folderState(folder).Search = ""
		s.folderState(folder).FilterActivity = ""
		s.folderState(folder).Filter = ""
	}
}

func FilterActivity(str string) SetStateFunc {
	return func(s *AccountState, folder string) {
		s.folderState(folder).FilterActivity = str
	}
}

func FilterResult(str string) SetStateFunc {
	return func(s *AccountState, folder string) {
		s.folderState(folder).FilterActivity = ""
		s.folderState(folder).Filter = concatFilters(s.folderState(folder).Filter, str)
	}
}

func concatFilters(existing, next string) string {
	if existing == "" {
		return next
	}
	return fmt.Sprintf("%s && %s", existing, next)
}

func Search(desc string) SetStateFunc {
	return func(s *AccountState, folder string) {
		s.folderState(folder).Search = desc
	}
}

func Sorting(on bool) SetStateFunc {
	return func(s *AccountState, folder string) {
		s.folderState(folder).Sorting = on
	}
}

func Threading(on bool) SetStateFunc {
	return func(s *AccountState, folder string) {
		s.folderState(folder).Threading = on
	}
}

func Passthrough(on bool) SetStateFunc {
	return func(s *AccountState, folder string) {
		s.passthrough = on
	}
}
