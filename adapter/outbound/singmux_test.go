package outbound

import (
	"strings"
	"testing"
	"time"

	"github.com/metacubex/mihomo/log"
)

func TestRefusalReporterWarnsOnceAMinute(t *testing.T) {
	sub := log.Subscribe()
	defer log.UnSubscribe(sub)

	report := refusalReporter("refusing node")
	report("lookup gone.example: no such host")
	report("lookup second.example: no such host")
	report("lookup third.example: no such host")

	var lines []log.Event
	timeout := time.After(300 * time.Millisecond)
collect:
	for {
		select {
		case event := <-sub:
			if strings.Contains(event.Payload, "refusing node") {
				lines = append(lines, event)
			}
		case <-timeout:
			break collect
		}
	}

	want := "[Mux] refusing node refused 1 stream(s), the last: lookup gone.example: no such host"
	if len(lines) != 1 || lines[0].Payload != want || lines[0].LogLevel != log.WARNING {
		t.Fatalf("got %v, want one warning %q", lines, want)
	}
}
