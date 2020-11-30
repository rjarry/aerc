package ui

import (
	"fmt"

	"git.sr.ht/~sircmpwn/aerc/config"

	"github.com/gdamore/tcell/v2"
)

type Stack struct {
	children     []Drawable
	onInvalidate []func(d Drawable)
	uiConfig     config.UIConfig
}

func NewStack(uiConfig config.UIConfig) *Stack {
	return &Stack{uiConfig: uiConfig}
}

func (stack *Stack) Children() []Drawable {
	return stack.children
}

func (stack *Stack) OnInvalidate(onInvalidate func(d Drawable)) {
	stack.onInvalidate = append(stack.onInvalidate, onInvalidate)
}

func (stack *Stack) Invalidate() {
	for _, fn := range stack.onInvalidate {
		fn(stack)
	}
}

func (stack *Stack) Draw(ctx *Context) {
	if len(stack.children) > 0 {
		stack.Peek().Draw(ctx)
	} else {
		ctx.Fill(0, 0, ctx.Width(), ctx.Height(), ' ',
			stack.uiConfig.GetStyle(config.STYLE_STACK))
	}
}

func (stack *Stack) MouseEvent(localX int, localY int, event tcell.Event) {
	if len(stack.children) > 0 {
		switch element := stack.Peek().(type) {
		case Mouseable:
			element.MouseEvent(localX, localY, event)
		}
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
