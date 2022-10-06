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
	atomic.StoreInt32(&dirty, DIRTY)
}
