package ui

type Drawable interface {
	// Called when this renderable should draw itself
	Draw(ctx *Context)
	// Specifies a function to call when this cell needs to be redrawn
	OnInvalidate(callback func(d Drawable))
	// Invalidates the drawable
	Invalidate()
}
