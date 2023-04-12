package panicrgoup

import (
	"context"
	"fmt"
	"testing"
	"time"
)

const (
	total = 100
)

// Goroutine A panic or error, doest not affect the Goroutine B.
func TestPanicGroup_ErrOrPanicNotAffect(t *testing.T) {
	var pg PanicGroup
	pg.Go(func() error {
		for i := 0; i < total; i++ {
			if i%10 == 0 {
				panic("Goroutine A panic")
			}
		}
		return nil
	})

	var bCount = 0
	pg.Go(func() error {
		for ; bCount < total; bCount++ {
			if bCount%10 == 0 {
				time.Sleep(time.Millisecond)
			}
		}
		return nil
	})

	if err := pg.Wait(Error); err != nil {
		pe, ok := err.(*PanicError)
		if ok {
			t.Logf("recover: %v", pe.R)
			t.Logf("stack: %v", string(pe.Stack))
		} else {
			t.Logf("Goroutine A failed, err: %v", err)
		}
	}
	if bCount != total {
		t.Errorf("expected %v, got %v", total, bCount)
	}
}

// Goroutine A panic or error, so that the Goroutine B also ends.
func TestPanicGroup_ErrOrPanicAffect(t *testing.T) {
	pg, ctx := WithContext(context.Background())

	funcA := func(ctx context.Context) error {
		for i := 0; i < total; i++ {
			if i%10 == 0 {
				panic("Goroutine A panic")
			}
		}
		return nil
	}

	pg.Go(func() error {
		return funcA(ctx)
	})

	var bCount = 0
	pg.Go(func() error {
		c1 := make(chan int, total)
		for ; bCount < total; bCount++ {
			c1 <- bCount
			select {
			case <-ctx.Done():
				fmt.Println("Goroutine B end...")
				return nil
			case cc := <-c1:
				if cc%10 == 0 {
					time.Sleep(time.Millisecond)
				}
			}
		}
		return nil
	})

	if err := pg.Wait(Error); err != nil {
		pe, ok := err.(*PanicError)
		if ok {
			t.Logf("recover: %v", pe.R)
			t.Logf("stack: %v", string(pe.Stack))
		} else {
			t.Logf("Goroutine A failed, err: %v", err)
		}
	}
	if bCount == total {
		t.Errorf("expected not equal %v, got %v", total, bCount)
	} else {
		t.Logf("Goroutine B count: %v", bCount)
	}
}

// Goroutine A get panic, let the main goroutine panic and catch the panic.
func TestPanicGroup_ErrOrPanicCatch(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			pe, ok := r.(*PanicError)
			if ok {
				t.Logf("main goroutine recover PanicError: %+v", pe)
			} else {
				t.Logf("main goroutine recover: %v", r)
			}
		}
	}()
	var pg PanicGroup
	pg.Go(func() error {
		for i := 0; i < total; i++ {
			if i%10 == 0 {
				panic("Goroutine A panic")
			}
		}
		return nil
	})

	var bCount = 0
	pg.Go(func() error {
		for ; bCount < total; bCount++ {
			if bCount%10 == 0 {
				time.Sleep(time.Millisecond)
			}
		}
		return nil
	})

	if err := pg.Wait(Panic); err != nil {
		pe, ok := err.(*PanicError)
		if ok {
			t.Logf("recover: %v", pe.R)
			t.Logf("stack: %v", string(pe.Stack))
		} else {
			t.Logf("Goroutine A failed, err: %v", err)
		}
	}
	if bCount != total {
		t.Errorf("expected %v, got %v", total, bCount)
	}
}
