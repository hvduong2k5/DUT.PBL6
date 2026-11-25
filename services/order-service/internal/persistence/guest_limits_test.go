package persistence

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestGuestPendingLimitSerializesConcurrentKeys(t *testing.T) {
	h := database(t, false)
	input := acceptanceInput(t, 1)
	p := &Payment{Store: h.store}
	now := time.Now()
	if e := p.CreateGuestSession(h.ctx, input.Owner.PrincipalID, "guest-credential", now, now.Add(time.Hour)); e != nil {
		t.Fatal(e)
	}
	var accepted, limited atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		in := acceptanceInput(t, 1)
		in.GuestPendingLimit = 3
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, e := h.store.Accept(h.ctx, in)
			if e == nil {
				accepted.Add(1)
			} else if errors.Is(e, ErrGuestPendingLimit) {
				limited.Add(1)
			} else {
				t.Error(e)
			}
		}()
	}
	wg.Wait()
	if accepted.Load() != 3 || limited.Load() != 5 || h.count(t, "SELECT count(*) FROM checkout_operations") != 3 {
		t.Fatal("concurrent guest quota", accepted.Load(), limited.Load())
	}
}
