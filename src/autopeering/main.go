package autopeering

import (
	"net"
	"net/url"
	"time"
)

const defaultTimeout time.Duration = time.Duration(3) * time.Second

type Peer struct {
	URL     url.URL
	Online  bool
	Latency time.Duration
}

func testPeers(peers []url.URL) []Peer {
	var res []Peer
	results := make(chan Peer)

	for _, p := range peers {
		go testPeer(p, results)
	}

	for i := 0; i < len(peers); i++ {
		res = append(res, <-results)
	}

	return res
}

func testPeer(peer url.URL, results chan Peer) {
	p := Peer{peer, false, 0.0}
	p.Online = false
	t0 := time.Now()

	var network string
	if peer.Scheme == "tcp" || peer.Scheme == "tls" {
		network = "tcp"
	} else { // skip, not supported yet
		results <- p
		return
	}

	conn, err := net.DialTimeout(network, peer.Host, defaultTimeout)
	if err == nil {
		t1 := time.Now()
		conn.Close()
		p.Online = true
		p.Latency = t1.Sub(t0)
	}
	results <- p
}
