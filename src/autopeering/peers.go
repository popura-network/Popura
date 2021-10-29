package autopeering

import (
	_ "embed"
	"math/rand"
	"net/url"
	"sort"
	"strings"
	"time"
)

//go:embed peers.txt
var PublicPeers string

// Get URLs of embedded public peers
func GetPublicPeers() []url.URL {
	var result []url.URL

	for _, p := range strings.Split(PublicPeers, "\n") {
		if url, err := url.Parse(p); err == nil {
			result = append(result, *url)
		} else {
			panic(err)
		}
	}
	return result
}

// Get n online peers with best latency from a peer list
func GetClosestPeers(peerList []url.URL, n int) []url.URL {
	var result []url.URL
	onlinePeers := testPeers(peerList)

	// Filter online peers
	x := 0
	for _, p := range onlinePeers {
		if p.Online {
			onlinePeers[x] = p
			x++
		}
	}
	onlinePeers = onlinePeers[:x]

	sort.Slice(onlinePeers, func(i, j int) bool {
		return onlinePeers[i].Latency < onlinePeers[j].Latency
	})

	for i := 0; i < len(onlinePeers); i++ {
		if len(result) == n {
			break
		}
		result = append(result, onlinePeers[i].URL)
	}

	return result
}

// Pick n random peers from a list
func RandomPick(peerList []url.URL, n int) []url.URL {
	if len(peerList) <= n {
		return peerList
	}

	var res []url.URL
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for _, i := range r.Perm(n) {
		res = append(res, peerList[i])
	}

	return res
}
