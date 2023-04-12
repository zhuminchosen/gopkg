package panicrgoup

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
)

// ErrType is used to indicate different ways of handling panics.
type ErrType int

const (
	// Error indicates the Wait(ErrType) function will return error whether it is
	// error or panic.
	Error ErrType = iota
	// Panic indicates the Wait(ErrType) function will panic when the error is
	// *PanicError, the main goroutine needs to catch panic by itself.
	Panic
)

// A PanicGroup is a collection of goroutines working on subtasks that are part of
// the same overall task.
//
// A zero PanicGroup is valid and does not cancel on error.
type PanicGroup struct {
	cancel func()

	wg sync.WaitGroup

	errOnce sync.Once
	err     error
}

type PanicError struct {
	R     interface{}
	Stack []byte
}

func (p *PanicError) Error() string {
	if p != nil {
		return fmt.Sprintf("R: %v \n Stack: %v ", p.R, string(p.Stack))
	}
	return ""
}

// WithContext returns a new PanicGroup and an associated Context derived from ctx.
//
// The derived Context is canceled the first time a function passed to Go
// returns a non-nil error or the first time Wait returns, whichever occurs
// first.
func WithContext(ctx context.Context) (*PanicGroup, context.Context) {
	ctx1, cancel := context.WithCancel(ctx)
	return &PanicGroup{cancel: cancel}, ctx1
}

// Wait blocks until all function calls from the Go method have returned, then
// returns the first non-nil error or panic (if any) from them.
func (g *PanicGroup) Wait(wo ErrType) error {
	g.wg.Wait()
	if g.cancel != nil {
		g.cancel()
	}

	if g.err == nil {
		return nil
	}
	switch wo {
	case Error:
		return g.err
	case Panic:
		pErr, ok := g.err.(*PanicError)
		if ok {
			panic(pErr)
		} else {
			return g.err
		}
	default:
		return g.err
	}

}

// Go calls the given function in a new goroutine.
//
// The first call to return a non-nil error cancels the group; its error will be
// returned by Wait.
func (g *PanicGroup) Go(f func() error) {
	g.wg.Add(1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				g.errOnce.Do(func() {
					g.err = &PanicError{
						R:     r,
						Stack: debug.Stack(),
					}
					if g.cancel != nil {
						g.cancel()
					}
				})
			}

			g.wg.Done()
		}()

		if err := f(); err != nil {
			g.errOnce.Do(func() {
				g.err = err
				if g.cancel != nil {
					g.cancel()
				}
			})
		}
	}()
}
