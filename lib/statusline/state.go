package statusline

import (
	"fmt"
	"strings"
)

type State struct {
	Name      string
	Multiple  bool
	Separator string

	Connection   string
	ConnActivity string
	Connected    bool

	Passthrough string

	fs map[string]*folderState
}

func NewState(name string, multipleAccts bool, sep string) *State {
	return &State{Name: name, Multiple: multipleAccts, Separator: sep,
		fs: make(map[string]*folderState)}
}

func (s *State) StatusLine(folder string) string {
	var line []string
	if s.Connection != "" || s.ConnActivity != "" {
		conn := s.Connection
		if s.ConnActivity != "" {
			conn = s.ConnActivity
		}
		if s.Multiple {
			line = append(line, fmt.Sprintf("[%s] %s", s.Name, conn))
		} else {
			line = append(line, conn)
		}
	}
	if s.Connected {
		if s.Passthrough != "" {
			line = append(line, s.Passthrough)
		}
		if folder != "" {
			line = append(line, s.folderState(folder).State()...)
		}
	}
	return strings.Join(line, s.Separator)
}

func (s *State) folderState(folder string) *folderState {
	if _, ok := s.fs[folder]; !ok {
		s.fs[folder] = &folderState{}
	}
	return s.fs[folder]
}

type SetStateFunc func(s *State, folder string)

func Connected(state bool) SetStateFunc {
	return func(s *State, folder string) {
		s.ConnActivity = ""
		s.Connected = state
		if state {
			s.Connection = "Connected"
		} else {
			s.Connection = "Disconnected"
		}
	}
}

func ConnectionActivity(desc string) SetStateFunc {
	return func(s *State, folder string) {
		s.ConnActivity = desc
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

func Threading(on bool) SetStateFunc {
	return func(s *State, folder string) {
		s.folderState(folder).Threading = ""
		if on {
			s.folderState(folder).Threading = "threading"
		}
	}
}

func Passthrough(on bool) SetStateFunc {
	return func(s *State, folder string) {
		s.Passthrough = ""
		if on {
			s.Passthrough = "passthrough"
		}
	}
}
