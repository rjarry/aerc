package ui

import (
	"fmt"

	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rockorager/vaxis"
)

type Stack struct {
	children []Drawable
	uiConfig *config.UIConfig
}

func NewStack(uiConfig *config.UIConfig) *Stack {
	return &Stack{uiConfig: uiConfig}
}

func (stack *Stack) Children() []Drawable {
	return stack.children
}

func (stack *Stack) Invalidate() {
	Invalidate()
}

func (stack *Stack) Draw(ctx *Context) {
	if len(stack.children) > 0 {
		stack.Peek().Draw(ctx)
	} else {
		ctx.Fill(0, 0, ctx.Width(), ctx.Height(), ' ',
			stack.uiConfig.GetStyle(config.STYLE_STACK))
	}
}

func (stack *Stack) MouseEvent(localX int, localY int, event vaxis.Event) {
	if len(stack.children) > 0 {
		if element, ok := stack.Peek().(Mouseable); ok {
			element.MouseEvent(localX, localY, event)
		}
	}
}

func (stack *Stack) Push(d Drawable) {
	stack.children = append(stack.children, d)
	stack.Invalidate()
}

func (stack *Stack) Pop() Drawable {
	if len(stack.children) == 0 {
		panic(fmt.Errorf("Tried to pop from an empty UI stack"))
	}
	d := stack.children[len(stack.children)-1]
	stack.children = stack.children[:len(stack.children)-1]
	stack.Invalidate()
	return d
}

func (stack *Stack) Peek() Drawable {
	if len(stack.children) == 0 {
		panic(fmt.Errorf("Tried to peek from an empty stack"))
	}
	return stack.children[len(stack.children)-1]
}
