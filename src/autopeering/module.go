package autopeering

import (
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/gologme/log"

	"github.com/yggdrasil-network/yggdrasil-go/src/admin"
	"github.com/yggdrasil-network/yggdrasil-go/src/config"
	"github.com/yggdrasil-network/yggdrasil-go/src/core"

	"github.com/popura-network/Popura/src/popura"
)

const (
	linkLocalPrefix  = "fe80"
	autopeerTimeout  = time.Minute
	peerCheckTimeout = 10 * time.Second
)

type AutoPeering struct {
	core           *core.Core
	log            *log.Logger
	checkPeerTimer *time.Timer
	hadPeers       time.Time
	proxyURL       *url.URL
	publicPeers    *[]string
}

func (ap *AutoPeering) Init(yggcore *core.Core, yggConfig *config.NodeConfig, popConfig *popura.PopuraConfig, log *log.Logger, options interface{}) error {
	ap.core = yggcore
	ap.log = log

	proxyEnv := os.Getenv("ALL_PROXY")
	if proxyEnv == "" {
		proxyEnv = os.Getenv("all_proxy")
	}

	if proxyEnv == "" {
		ap.publicPeers = &PublicPeers
	} else {
		tcpPeers := GetTcpPeers()
		ap.publicPeers = &tcpPeers
	}

	var err error
	ap.proxyURL, err = url.Parse(proxyEnv)
	return err
}

func (ap *AutoPeering) Start() error {
	go ap.checkPeerLoop()
	ap.log.Infoln("autopeering: module started")
	return nil
}

func (ap *AutoPeering) Stop() error {
	if ap.checkPeerTimer != nil {
		ap.checkPeerTimer.Stop()
	}
	return nil
}

func (ap *AutoPeering) checkPeerLoop() {
	havePeers := false

	for _, p := range ap.core.GetPeers() {
		if p.Remote[:4] != linkLocalPrefix {
			havePeers = true
			break
		}
	}

	if havePeers {
		ap.hadPeers = time.Now()
	} else if time.Since(ap.hadPeers) > autopeerTimeout {
		ap.hadPeers = time.Now()
		peers := RandomPick(GetClosestPeers(*ap.publicPeers, 10), 1)
		if len(peers) == 1 {
			peerUri := ap.getPeerUri(peers[0])

			ap.log.Infoln("autopeering: adding new peer", peerUri)
			if err := ap.core.CallPeer(peerUri, ""); err != nil {
				ap.log.Infoln("autopeering: Failed to connect to peer:", err)
			}
		}
	}

	ap.checkPeerTimer = time.AfterFunc(peerCheckTimeout, func() {
		ap.checkPeerLoop()
	})
}

// Return peer URI with respect to proxy environment settings
func (ap *AutoPeering) getPeerUri(uriString string) *url.URL {
	if ap.proxyURL.IsAbs() {
		uriString = fmt.Sprintf("socks://%s/%s", ap.proxyURL.Host, uriString[6:len(uriString)])
	}

	uri, _ := url.Parse(uriString)

	return uri
}

func (ap *AutoPeering) UpdateConfig(yggConfig *config.NodeConfig, popConfig *popura.PopuraConfig) {}
func (ap *AutoPeering) SetupAdminHandlers(a *admin.AdminSocket)                                   {}
func (ap *AutoPeering) IsStarted() bool                                                           { return false }
