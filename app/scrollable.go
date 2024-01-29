package app

// Scrollable implements vertical scrolling
type Scrollable struct {
	scroll int
	offset int
	height int
	elems  int
}

func (s *Scrollable) Scroll() int {
	return s.scroll
}

func (s *Scrollable) SetOffset(offset int) {
	s.offset = offset
}

func (s *Scrollable) ScrollOffset() int {
	return s.offset
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

func (s *Scrollable) EnsureScroll(idx int) {
	if idx < 0 {
		return
	}

	middle := s.height / 2
	switch {
	case s.offset > middle:
		s.scroll = idx - middle
	case idx < s.scroll+s.offset:
		s.scroll = idx - s.offset
	case idx >= s.scroll-s.offset+s.height:
		s.scroll = idx + s.offset - s.height + 1
	}

	s.checkBounds()
}

func (s *Scrollable) checkBounds() {
	maxScroll := s.elems - s.height
	if maxScroll < 0 {
		maxScroll = 0
	}

	if s.scroll > maxScroll {
		s.scroll = maxScroll
	}

	if s.scroll < 0 {
		s.scroll = 0
	}
}

type AlignPosition uint

const (
	AlignTop AlignPosition = iota
	AlignCenter
	AlignBottom
)

func (s *Scrollable) Align(idx int, pos AlignPosition) {
	switch pos {
	case AlignTop:
		s.scroll = idx
	case AlignCenter:
		s.scroll = idx - s.height/2
	case AlignBottom:
		s.scroll = idx - s.height + 1
	}
	s.checkBounds()
}
