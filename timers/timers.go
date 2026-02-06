package timers

import (
	"time"
	"context"
)

func SleepWithContext(d time.Duration, ctx context.Context) {
	select {
	case <-ctx.Done():
		return
	case <-time.After(d):
	}
}

type ResettableTimer struct {
	timer    *time.Timer
	fn       func()
	dur      time.Duration
	parent   context.Context
	ctx      context.Context
	cancelFn context.CancelFunc
	resetCh  chan bool
}

func New(ctx context.Context, d time.Duration, triggerNow bool, fn func()) *ResettableTimer {
	my_ctx, cancel := context.WithCancel(ctx)
	rt := &ResettableTimer{
		timer:    time.NewTimer(d),
		fn:       fn,
		dur:      d,
		parent:   ctx,
		ctx:      my_ctx,
		cancelFn: cancel,
		resetCh:  make(chan bool, 1),
	}

	go rt.run()

	if triggerNow {
		go fn()
	}

	return rt
}

func (rt *ResettableTimer) Reset() {
	select {
	case rt.resetCh <- true:
	default: // avoid blocking if reset already pending
	}
}

func (rt *ResettableTimer) Restart() {
	// Stop and drain the timer
	if !rt.timer.Stop() {
		select {
		case <-rt.timer.C:
		default:
		}
	}

	// Clear any pending resets
	for drained := false; !drained; {
		select {
		case <-rt.resetCh:
		default:
			drained = true
		}
	}

	// Signal the run loop to reset
	rt.resetCh <- true
	
	// Reset contexts
	rt.ctx, rt.cancelFn = context.WithCancel(rt.parent)
	go rt.run()
}


func (rt *ResettableTimer) Stop() {
	rt.cancelFn()
	rt.timer.Stop()
}

func (rt *ResettableTimer) run() {
	loop: for {
		select {
		case <-rt.timer.C:
			rt.fn()
		case <-rt.resetCh:
			for drained := false; !drained; {
				select {
				case <-rt.resetCh:
				default:
					drained = true
				}
			}
			if !rt.timer.Stop() {
				select {
				case <-rt.timer.C:
				default:
				}
			}
			rt.timer.Reset(rt.dur)
		case <-rt.ctx.Done():
			rt.timer.Stop()
			break loop
		}
	}
}
