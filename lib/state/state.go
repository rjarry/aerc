package state

import (
	"fmt"

	"git.sr.ht/~rjarry/aerc/config"
)

type State struct {
	renderer renderFunc
	acct     *accountState
	fldr     map[string]*folderState
	width    int
}

type accountState struct {
	Name         string
	Multiple     bool
	ConnActivity string
	Connected    bool
	Passthrough  bool
}

type folderState struct {
	Name           string
	Search         string
	Filter         string
	FilterActivity string
	Sorting        bool
	Threading      bool
}

func NewState(name string, multipleAccts bool) *State {
	return &State{
		renderer: newRenderer(),
		acct:     &accountState{Name: name, Multiple: multipleAccts},
		fldr:     make(map[string]*folderState),
	}
}

func (s *State) StatusLine(folder string) string {
	return s.renderer(renderParams{
		width: s.width,
		sep:   config.Statusline.Separator,
		acct:  s.acct,
		fldr:  s.folderState(folder),
	})
}

func (s *State) folderState(folder string) *folderState {
	if _, ok := s.fldr[folder]; !ok {
		s.fldr[folder] = &folderState{Name: folder}
	}
	return s.fldr[folder]
}

func (s *State) SetWidth(w int) bool {
	changeState := false
	if s.width != w {
		s.width = w
		changeState = true
	}
	return changeState
}

func (s *State) Connected() bool {
	return s.acct.Connected
}

type SetStateFunc func(s *State, folder string)

func SetConnected(state bool) SetStateFunc {
	return func(s *State, folder string) {
		s.acct.ConnActivity = ""
		s.acct.Connected = state
	}
}

func ConnectionActivity(desc string) SetStateFunc {
	return func(s *State, folder string) {
		s.acct.ConnActivity = desc
	}
}

func SearchFilterClear() SetStateFunc {
	return func(s *State, folder string) {
		s.folderState(folder).Search = ""
		s.folderState(folder).FilterActivity = ""
		s.folderState(folder).Filter = ""
	}
}

func FilterActivity(str string) SetStateFunc {
	return func(s *State, folder string) {
		s.folderState(folder).FilterActivity = str
	}
}

func FilterResult(str string) SetStateFunc {
	return func(s *State, folder string) {
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
	return func(s *State, folder string) {
		s.folderState(folder).Search = desc
	}
}

func Sorting(on bool) SetStateFunc {
	return func(s *State, folder string) {
		s.folderState(folder).Sorting = on
	}
}

func Threading(on bool) SetStateFunc {
	return func(s *State, folder string) {
		s.folderState(folder).Threading = on
	}
}

func Passthrough(on bool) SetStateFunc {
	return func(s *State, folder string) {
		s.acct.Passthrough = on
	}
}
