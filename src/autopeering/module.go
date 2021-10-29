package autopeering

import (
	"net/url"
	"strings"
	"time"

	"github.com/gologme/log"

	"github.com/yggdrasil-network/yggdrasil-go/src/admin"
	"github.com/yggdrasil-network/yggdrasil-go/src/config"
	"github.com/yggdrasil-network/yggdrasil-go/src/core"

	"github.com/popura-network/Popura/src/popura"
)

const (
	linkLocalPrefix  = "tls://[fe80"
	autopeerTimeout  = 30 * time.Second
	peerCheckTimeout = 10 * time.Second
)

type AutoPeering struct {
	core           *core.Core
	log            *log.Logger
	checkPeerTimer *time.Timer
	hadPeers       time.Time
	peers          []url.URL
}

func (ap *AutoPeering) Init(yggcore *core.Core, yggConfig *config.NodeConfig, popConfig *popura.PopuraConfig, log *log.Logger, options interface{}) error {
	ap.core = yggcore
	ap.log = log
	ap.peers = GetPublicPeers()
	return nil
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
		if !strings.HasPrefix(p.Remote, linkLocalPrefix) {
			ap.log.Debugln("autopeering: remote peer is connected ", p.Remote)
			havePeers = true
			break
		}
	}

	if havePeers {
		ap.hadPeers = time.Now()
	} else if time.Since(ap.hadPeers) > autopeerTimeout {
		ap.log.Debugln("autopeering: adding a new peer")
		ap.hadPeers = time.Now()
		peers := RandomPick(GetClosestPeers(ap.peers, 10), 1)
		if len(peers) == 1 {
			peerUri := peers[0]

			ap.log.Infoln("autopeering: adding new peer", peerUri.String())
			go func() {
				if err := ap.core.CallPeer(&peerUri, ""); err != nil {
					ap.log.Infoln("autopeering: peer connection failed:", err)
				}
			}()
		}
	}

	ap.checkPeerTimer = time.AfterFunc(peerCheckTimeout, func() {
		ap.checkPeerLoop()
	})
}

func (ap *AutoPeering) UpdateConfig(yggConfig *config.NodeConfig, popConfig *popura.PopuraConfig) {}
func (ap *AutoPeering) SetupAdminHandlers(a *admin.AdminSocket)                                   {}
func (ap *AutoPeering) IsStarted() bool                                                           { return false }
