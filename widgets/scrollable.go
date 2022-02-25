package widgets

// Scrollable implements vertical scrolling
type Scrollable struct {
	scroll int
	height int
	elems  int
}

func (s *Scrollable) Scroll() int {
	return s.scroll
}

func (s *Scrollable) PercentVisible() float64 {
	if s.elems <= 0 {
		return 1.0
	}
	return float64(s.height) / float64(s.elems)
}

func (s *Scrollable) PercentScrolled() float64 {
	if s.elems <= 0 {
		return 1.0
	}
	return float64(s.scroll) / float64(s.elems)
}

func (s *Scrollable) NeedScrollbar() bool {
	needScrollbar := true
	if s.PercentVisible() >= 1.0 {
		needScrollbar = false
	}
	return needScrollbar
}

func (s *Scrollable) UpdateScroller(height, elems int) {
	s.height = height
	s.elems = elems
}

func (s *Scrollable) EnsureScroll(selectingIdx int) {
	if selectingIdx < 0 {
		return
	}

	maxScroll := s.elems - s.height
	if maxScroll < 0 {
		maxScroll = 0
	}

	if selectingIdx >= s.scroll && selectingIdx < s.scroll+s.height {
		if s.scroll > maxScroll {
			s.scroll = maxScroll
		}
		return
	}

	if selectingIdx >= s.scroll+s.height {
		s.scroll = selectingIdx - s.height + 1
	} else if selectingIdx < s.scroll {
		s.scroll = selectingIdx
	}

	if s.scroll > maxScroll {
		s.scroll = maxScroll
	}

}
