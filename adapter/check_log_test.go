package adapter

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/metacubex/mihomo/adapter/outbound"
	"github.com/metacubex/mihomo/log"
)

func TestLogCheckOnlyOnChange(t *testing.T) {
	sub := log.Subscribe()
	defer log.UnSubscribe(sub)

	p := NewProxy(outbound.NewDirect())
	timeout := errors.New("i/o timeout")
	p.logCheck(true, 120, nil)             // answering from the start: nothing
	p.logCheck(false, 0, timeout)          // stopped
	p.logCheck(false, 0, timeout)          // still silent: nothing
	p.logCheck(true, 0, context.Canceled)  // cut short: nothing, and not counted as an answer
	p.logCheck(false, 0, context.Canceled) // cut short: nothing
	p.logCheck(true, 180, nil)             // answers again
	p.logCheck(true, 175, nil)             // still answering: nothing

	want := []string{
		"[Check] DIRECT stopped answering: i/o timeout",
		"[Check] DIRECT answers again, 180 ms",
	}
	for _, w := range want {
		select {
		case e := <-sub:
			if e.Payload != w {
				t.Fatalf("got %q, want %q", e.Payload, w)
			}
		case <-time.After(time.Second):
			t.Fatalf("no line %q", w)
		}
	}
	select {
	case e := <-sub:
		t.Fatalf("extra line %q", e.Payload)
	case <-time.After(100 * time.Millisecond):
	}
}
