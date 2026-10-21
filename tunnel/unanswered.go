package tunnel

import (
	"sync"
	"time"

	C "github.com/metacubex/mihomo/constant"
	"github.com/metacubex/mihomo/log"
	"github.com/metacubex/mihomo/tunnel/statistic"
)

// answers counts, per node, the TCP connections that sent something and got nothing back, out of
// all its TCP connections. A node that answers a mux stream before dialing it, as Xray's does,
// closes the stream with no byte back when it cannot reach the site, and nothing else logs that.
// At most one warning a minute per node, so that dead ad domains do not flood the log; the total
// tells a dead node from them.
//
// ponytail: unanswered connections after the last warning of a burst wait for the next one to be
// reported, as the mux refusal warning does; nodes of a replaced config stay in the map.
var answers = struct {
	sync.Mutex
	nodes map[string]*nodeAnswers
}{nodes: map[string]*nodeAnswers{}}

type nodeAnswers struct {
	closed, unanswered int
	reported           time.Time
}

// countAnswer is called once a connection's relay has ended, when its counters are final.
func countAnswer(info *statistic.TrackerInfo) {
	node := info.Chain.Last()
	if !isNode(node) {
		return
	}
	unanswered := info.UploadTotal.Load() > 0 && info.DownloadTotal.Load() == 0

	answers.Lock()
	defer answers.Unlock()
	n := answers.nodes[node]
	if n == nil {
		n = &nodeAnswers{}
		answers.nodes[node] = n
	}
	n.closed++
	if !unanswered {
		return
	}
	n.unanswered++
	if time.Since(n.reported) < time.Minute {
		return
	}
	log.Warnln("[TCP] %s: no answer on %d of %d connection(s), the last to %s", node, n.unanswered, n.closed, info.Metadata.RemoteAddress())
	n.closed, n.unanswered, n.reported = 0, 0, time.Now()
}

// isNode tells a proxy from the core's own outbounds, by type, since a config may rename those
// (our template's DIRECTv4). A node from a proxy provider is not in the map and counts.
func isNode(name string) bool {
	configMux.RLock()
	proxy, ok := proxies[name]
	configMux.RUnlock()
	if !ok {
		return true
	}
	switch proxy.Type() {
	case C.Direct, C.Reject, C.RejectDrop, C.Compatible, C.Pass, C.PassRule, C.Rematch, C.Dns:
		return false
	}
	return true
}
