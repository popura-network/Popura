package radv

import (
	"math/rand"
	"net"
	"time"

	"github.com/gologme/log"
	"github.com/mdlayher/ndp"

	"github.com/yggdrasil-network/yggdrasil-go/src/admin"
	"github.com/yggdrasil-network/yggdrasil-go/src/config"
	"github.com/yggdrasil-network/yggdrasil-go/src/core"

	"github.com/popura-network/Popura/src/popura"
)

var yggdrasilPrefixIP net.IP = net.ParseIP("200::")

const (
	maxInitialAdvInterval = 16 * time.Second
	maxInitialAdv         = 3
	minDelay              = 200 * time.Second
	maxDelay              = 600 * time.Second
)

type RAdv struct {
	log     *log.Logger
	conn    *ndp.Conn
	config  popura.RAdvConfig
	message *ndp.RouterAdvertisement
	subnet  *net.IPNet
	quit    chan struct{}
}

func (s *RAdv) Init(yggcore *core.Core, yggConfig *config.NodeConfig, popConfig *popura.PopuraConfig, log *log.Logger, options interface{}) error {
	s.log = log
	s.subnet = popura.SubnetFromKey(yggConfig.PrivateKey)
	s.config = popConfig.RAdv
	s.quit = make(chan struct{}, 2)

	return nil
}

func (s *RAdv) Start() error {
	if s.config.Enable {
		ifi, err := net.InterfaceByName(s.config.Interface)
		if err != nil {
			return err
		}
		s.checkForwardingEnabled()

		var ip net.IP
		s.conn, ip, err = ndp.Dial(ifi, ndp.LinkLocal)
		if err != nil {
			return err
		}

		if err := s.conn.JoinGroup(net.IPv6linklocalallrouters); err != nil {
			return err
		}

		if s.config.SetGatewayIP {
			if err := s.setGatewayIP(); err != nil {
				s.log.Errorln(err)
			}
		}

		routerLifetime := time.Second * 0
		options := []ndp.Option{
			&ndp.PrefixInformation{
				PrefixLength:                   64,
				OnLink:                         true,
				AutonomousAddressConfiguration: true,
				ValidLifetime:                  time.Second * 86400,
				PreferredLifetime:              time.Second * 14400,
				Prefix:                         s.subnet.IP,
			},
			&ndp.LinkLayerAddress{Direction: ndp.Source, Addr: ifi.HardwareAddr},
		}

		if s.config.DefaultRouter {
			routerLifetime = time.Second * 1800
		} else {
			options = append(options, &ndp.RouteInformation{
				PrefixLength:  7,
				Preference:    ndp.Medium,
				RouteLifetime: time.Second * 1800,
				Prefix:        yggdrasilPrefixIP,
			})
		}

		if s.config.DNS {
			options = append(options, &ndp.RecursiveDNSServer{
				Lifetime: time.Second * 4294967295,
				Servers:  []net.IP{ip},
			})
		}

		s.message = &ndp.RouterAdvertisement{
			CurrentHopLimit:           64,
			ManagedConfiguration:      false,
			OtherConfiguration:        false,
			RouterSelectionPreference: ndp.Medium,
			RouterLifetime:            routerLifetime,
			ReachableTime:             time.Second * 0,
			RetransmitTimer:           time.Second * 0,
			Options:                   options,
		}

		advTrigger := make(chan struct{})
		go s.listener(advTrigger)
		go s.advertiserTask(advTrigger)
		go s.multicast(advTrigger)

		s.log.Infof("Started RAdv on: %s%%%s", ip.String(), s.config.Interface)
	}
	return nil
}

// Send RA messages when triggered
func (s *RAdv) advertiserTask(advTrigger chan struct{}) {
	for {
		select {
		case <-s.quit:
			return
		case <-advTrigger:
			err := s.conn.WriteTo(s.message, nil, net.IPv6linklocalallnodes)
			if err != nil {
				s.log.Debugln(err)
			}
		}
	}
}

// Listen to Router Solicitation messages to trigger RAs
func (s *RAdv) listener(advTrigger chan struct{}) {
	for {
		select {
		case <-s.quit:
			return
		default:
			msg, _, _, err := s.conn.ReadFrom()
			if err != nil {
				s.log.Debug(err)
				continue
			}
			if _, ok := msg.(*ndp.RouterSolicitation); ok {
				advTrigger <- struct{}{}
			}
		}
	}
}

// Trigger RAs periodically
func (s *RAdv) multicast(advTrigger chan struct{}) {
	prng := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := 0; ; i++ {
		advTrigger <- struct{}{}

		select {
		case <-s.quit:
			return
		case <-time.After(multicastDelay(prng, i)):
		}
	}
}

func multicastDelay(r *rand.Rand, i int) time.Duration {
	// Implements the algorithm described in:
	// https://tools.ietf.org/html/rfc4861#section-6.2.4.

	var d time.Duration
	if minDelay == maxDelay {
		// Identical minDelay/maxDelay, use a static interval.
		d = (time.Duration(maxDelay) * time.Nanosecond).Round(time.Second)
	} else {
		// minDelay <= wait <= maxDelay, rounded to 1 second granularity.
		d = (minDelay + time.Duration(
			r.Int63n(maxDelay.Nanoseconds()-minDelay.Nanoseconds()),
		)*time.Nanosecond).Round(time.Second)
	}

	// For first few advertisements, select a shorter wait time so routers
	// can be discovered quickly, per the RFC.
	if i < maxInitialAdv && d > maxInitialAdvInterval {
		d = maxInitialAdvInterval
	}

	return d
}

func (s *RAdv) Stop() error {
	close(s.quit)

	if s.conn != nil {
		s.conn.Close()
	}

	if err := s.removeGatewayIP(); err != nil {
		s.log.Errorln(err)
	}
	return nil
}

func (s *RAdv) UpdateConfig(yggConfig *config.NodeConfig, popConfig *popura.PopuraConfig) {
	s.Stop()
	s.subnet = popura.SubnetFromKey(yggConfig.PrivateKey)
	s.config = popConfig.RAdv
	s.quit = make(chan struct{}, 2)
	if err := s.Start(); err != nil {
		s.log.Errorln("An error occured starting RAdv: ", err)
	}
}

func (s *RAdv) SetupAdminHandlers(a *admin.AdminSocket) {}

func (s *RAdv) IsStarted() bool {
	return false
}
