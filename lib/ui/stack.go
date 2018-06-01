package ui

import (
	"fmt"

	"github.com/gdamore/tcell"
)

type Stack struct {
	children []Drawable
	onInvalidate func(d Drawable)
}

func NewStack() *Stack {
	return &Stack{}
}

func (stack *Stack) OnInvalidate(onInvalidate func (d Drawable)) {
	stack.onInvalidate = onInvalidate
}

func (stack *Stack) Invalidate() {
	if stack.onInvalidate != nil {
		stack.onInvalidate(stack)
	}
}

func (stack *Stack) Draw(ctx *Context) {
	if len(stack.children) > 0 {
		stack.Peek().Draw(ctx)
	} else {
		ctx.Fill(0, 0, ctx.Width(), ctx.Height(), ' ', tcell.StyleDefault)
	}
}

func (stack *Stack) Push(d Drawable) {
	if len(stack.children) != 0 {
		stack.Peek().OnInvalidate(nil)
	}
	stack.children = append(stack.children, d)
	d.OnInvalidate(stack.invalidateFromChild)
	stack.Invalidate()
}

func (stack *Stack) Pop() Drawable {
	if len(stack.children) == 0 {
		panic(fmt.Errorf("Tried to pop from an empty UI stack"))
	}
	d := stack.children[len(stack.children)-1]
	stack.children = stack.children[:len(stack.children)-1]
	stack.Invalidate()
	d.OnInvalidate(nil)
	if len(stack.children) != 0 {
		stack.Peek().OnInvalidate(stack.invalidateFromChild)
	}
	return d
}

func (stack *Stack) Peek() Drawable {
	if len(stack.children) == 0 {
		panic(fmt.Errorf("Tried to peek from an empty stack"))
	}
	return stack.children[len(stack.children)-1]
}

func (stack *Stack) invalidateFromChild(d Drawable) {
	stack.Invalidate()
}
