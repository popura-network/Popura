package autopeering

import (
	"math/rand"
	"net"
	"sort"
	"strings"
	"time"
)

const defaultTimeout time.Duration = time.Duration(3) * time.Second

type Peer struct {
	URI     string
	Online  bool
	Latency time.Duration
}

func testPeers(peers []string) []Peer {
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

func testPeer(pstring string, results chan Peer) {
	p := Peer{pstring, false, 0.0}
	p.Online = false
	t0 := time.Now()
	peer_addr := strings.Split(p.URI, "://")
	if len(peer_addr) != 2 {
		results <- p
		return
	}
	var network string
	if peer_addr[0] == "tcp" || peer_addr[0] == "tls" {
		network = "tcp"
	} else { // skip, not supported yet
		results <- p
		return
	}

	conn, err := net.DialTimeout(network, peer_addr[1], defaultTimeout)
	if err == nil {
		t1 := time.Now()
		conn.Close()
		p.Online = true
		p.Latency = t1.Sub(t0)
	}
	results <- p
}

// Get X online peers with best latency from a peer list
func GetClosestPeers(peerList []string, num int) []string {
	var res []string
	testedPeers := testPeers(peerList)

	// Filter online peers
	n := 0
	for _, x := range testedPeers {
		if x.Online == true {
			testedPeers[n] = x
			n++
		}
	}
	testedPeers = testedPeers[:n]

	sort.Slice(testedPeers, func(i, j int) bool {
		return testedPeers[i].Latency < testedPeers[j].Latency
	})

	for i := 0; i < len(testedPeers); i++ {
		if len(res) == num {
			break
		}
		res = append(res, testedPeers[i].URI)
	}

	return res
}

// Pick num random peers from a list
func RandomPick(peerList []string, num int) []string {
	if len(peerList) <= num {
		return peerList
	}

	var res []string
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for _, i := range r.Perm(num) {
		res = append(res, peerList[i])
	}

	return res
}
