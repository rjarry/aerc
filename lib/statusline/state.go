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

	Search         string
	Filter         string
	FilterActivity string

	Threading   string
	Passthrough string
}

func NewState(name string, multipleAccts bool, sep string) *State {
	return &State{Name: name, Multiple: multipleAccts, Separator: sep}
}

func (s *State) String() string {
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
		if s.FilterActivity != "" {
			line = append(line, s.FilterActivity)
		} else {
			if s.Filter != "" {
				line = append(line, s.Filter)
			}
		}
		if s.Search != "" {
			line = append(line, s.Search)
		}
		if s.Threading != "" {
			line = append(line, s.Threading)
		}
		if s.Passthrough != "" {
			line = append(line, s.Passthrough)
		}
	}
	return strings.Join(line, s.Separator)
}

type SetStateFunc func(s *State)

func Connected(state bool) SetStateFunc {
	return func(s *State) {
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
	return func(s *State) {
		s.ConnActivity = desc
	}
}

func SearchFilterClear() SetStateFunc {
	return func(s *State) {
		s.Search = ""
		s.FilterActivity = ""
		s.Filter = ""
	}
}

func FilterActivity(str string) SetStateFunc {
	return func(s *State) {
		s.FilterActivity = str
	}
}

func FilterResult(str string) SetStateFunc {
	return func(s *State) {
		s.FilterActivity = ""
		s.Filter = concatFilters(s.Filter, str)
	}
}

func concatFilters(existing, next string) string {
	if existing == "" {
		return next
	}
	return fmt.Sprintf("%s && %s", existing, next)
}

func Search(desc string) SetStateFunc {
	return func(s *State) {
		s.Search = desc
	}
}

func Threading(on bool) SetStateFunc {
	return func(s *State) {
		s.Threading = ""
		if on {
			s.Threading = "threading"
		}
	}
}

func Passthrough(on bool) SetStateFunc {
	return func(s *State) {
		s.Passthrough = ""
		if on {
			s.Passthrough = "passthrough"
		}
	}
}
