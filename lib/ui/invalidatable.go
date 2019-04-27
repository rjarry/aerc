package ui

import (
	"sync/atomic"
)

type Invalidatable struct {
	onInvalidate atomic.Value
}

func (i *Invalidatable) OnInvalidate(f func(d Drawable)) {
	i.onInvalidate.Store(f)
}

func (i *Invalidatable) DoInvalidate(d Drawable) {
	v := i.onInvalidate.Load()
	if v == nil {
		return
	}
	f := v.(func(d Drawable))
	if f != nil {
		f(d)
	}
}
