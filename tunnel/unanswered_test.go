package tunnel

import (
	"strings"
	"testing"
	"time"

	"github.com/metacubex/mihomo/common/atomic"
	C "github.com/metacubex/mihomo/constant"
	"github.com/metacubex/mihomo/log"
	"github.com/metacubex/mihomo/tunnel/statistic"
)

// typedProxy stands in for an outbound: countAnswer asks only its type. adapter/outbound would
// import this package back.
type typedProxy struct {
	C.Proxy
	tp C.AdapterType
}

func (p typedProxy) Type() C.AdapterType { return p.tp }

func closedOn(network C.NetWork, node, host string, upload, download int64) {
	countAnswer(&statistic.TrackerInfo{
		Metadata:      &C.Metadata{NetWork: network, Host: host, DstPort: 443},
		UploadTotal:   atomic.NewInt64(upload),
		DownloadTotal: atomic.NewInt64(download),
		Chain:         C.Chain{node, "Main"},
	})
}

func TestCountAnswerWarnsOnceAMinutePerNode(t *testing.T) {
	configMux.Lock()
	saved := proxies
	proxies = map[string]C.Proxy{"DIRECTv4": typedProxy{tp: C.Direct}}
	configMux.Unlock()
	defer func() {
		configMux.Lock()
		proxies = saved
		configMux.Unlock()
	}()

	sub := log.Subscribe()
	defer log.UnSubscribe(sub)

	closed := func(node, host string, upload, download int64) {
		closedOn(C.TCP, node, host, upload, download)
	}
	closed("silent node", "answered.example", 517, 4096)
	closed("silent node", "gone.example", 517, 0)
	closed("silent node", "second.example", 517, 0)
	closed("silent node", "never-sent.example", 0, 0)
	closed("DIRECTv4", "blocked.example", 517, 0)

	var lines []log.Event
	timeout := time.After(300 * time.Millisecond)
collect:
	for {
		select {
		case event := <-sub:
			if strings.HasPrefix(event.Payload, "[TCP]") {
				lines = append(lines, event)
			}
		case <-timeout:
			break collect
		}
	}

	want := "[TCP] silent node: no answer on 1 of 2 connection(s), the last to gone.example:443"
	if len(lines) != 1 || lines[0].Payload != want || lines[0].LogLevel != log.WARNING {
		t.Fatalf("got %v, want one warning %q", lines, want)
	}
}

// UDP sessions of a node are counted apart from its TCP connections: a node whose UDP is broken shows as all of its
// sessions unanswered, while its TCP keeps answering.
func TestCountAnswerKeepsUDPApart(t *testing.T) {
	sub := log.Subscribe()
	defer log.UnSubscribe(sub)

	closedOn(C.TCP, "udp-broken node", "site.example", 517, 4096)
	closedOn(C.UDP, "udp-broken node", "call.example", 120, 0)

	var lines []string
	timeout := time.After(300 * time.Millisecond)
collect:
	for {
		select {
		case event := <-sub:
			if strings.Contains(event.Payload, "udp-broken node") {
				lines = append(lines, event.Payload)
			}
		case <-timeout:
			break collect
		}
	}

	want := "[UDP] udp-broken node: no answer on 1 of 1 connection(s), the last to call.example:443"
	if len(lines) != 1 || lines[0] != want {
		t.Fatalf("got %q, want only %q", lines, want)
	}
}
