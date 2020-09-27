package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/gologme/log"
	gsyslog "github.com/hashicorp/go-syslog"
	"github.com/kardianos/minwinsvc"

	"github.com/yggdrasil-network/yggdrasil-go/src/address"
	"github.com/yggdrasil-network/yggdrasil-go/src/admin"
	"github.com/yggdrasil-network/yggdrasil-go/src/config"
	"github.com/yggdrasil-network/yggdrasil-go/src/crypto"
	"github.com/yggdrasil-network/yggdrasil-go/src/module"
	"github.com/yggdrasil-network/yggdrasil-go/src/multicast"
	"github.com/yggdrasil-network/yggdrasil-go/src/tuntap"
	"github.com/yggdrasil-network/yggdrasil-go/src/version"
	"github.com/yggdrasil-network/yggdrasil-go/src/yggdrasil"

	_meshname "github.com/zhoreeq/meshname/pkg/meshname"

	"github.com/popura-network/Popura/src/autopeering"
	"github.com/popura-network/Popura/src/dhtcrawler"
	"github.com/popura-network/Popura/src/meshname"
	"github.com/popura-network/Popura/src/radv"
	"github.com/popura-network/Popura/src/popura"
)

type node struct {
	core      yggdrasil.Core
	state     *config.NodeState
	tuntap    module.Module // tuntap.TunAdapter
	multicast module.Module // multicast.Multicast
	admin     module.Module // admin.AdminSocket
	meshname  popura.Module // meshname.MeshnameServer
	radv      popura.Module // radv.RAdv
	autopeering      popura.Module // autopeering.AutoPeering
}

func setLogLevel(loglevel string, logger *log.Logger) {
	levels := [...]string{"error", "warn", "info", "debug", "trace"}
	loglevel = strings.ToLower(loglevel)

	contains := func() bool {
		for _, l := range levels {
			if l == loglevel {
				return true
			}
		}
		return false
	}

	if !contains() { // set default log level
		logger.Infoln("Loglevel parse failed. Set default level(info)")
		loglevel = "info"
	}

	for _, l := range levels {
		logger.EnableLevel(l)
		if l == loglevel {
			break
		}
	}
}

// The main function is responsible for configuring and starting Yggdrasil.
func run_yggdrasil() {
	// Configure the command line parameters.
	genconf := flag.Bool("genconf", false, "print a new config to stdout")
	useconf := flag.Bool("useconf", false, "read HJSON/JSON config from stdin")
	useconffile := flag.String("useconffile", "", "read HJSON/JSON config from specified file path")
	normaliseconf := flag.Bool("normaliseconf", false, "use in combination with either -useconf or -useconffile, outputs your configuration normalised")
	confjson := flag.Bool("json", false, "print configuration from -genconf or -normaliseconf as JSON instead of HJSON")
	autoconf := flag.Bool("autoconf", false, "automatic mode (dynamic IP, peer with IPv6 neighbors)")
	autopeer := flag.Bool("autopeer", false, "automatic Internet peering (using peers from github.com/yggdrasil-network/public-peers)")
	ver := flag.Bool("version", false, "prints the version of this build")
	logto := flag.String("logto", "stdout", "file path to log to, \"syslog\" or \"stdout\"")
	getaddr := flag.Bool("address", false, "returns the IPv6 address as derived from the supplied configuration")
	getsnet := flag.Bool("subnet", false, "returns the IPv6 subnet as derived from the supplied configuration")
	meshnameconf := flag.String("meshnameconf", "", "prints example Meshname.Config config value for a specified IP address")
	dhtcrawlenable := flag.Bool("dhtcrawler", false, "Enable getDHTCrawl AdminAPI method")
	loglevel := flag.String("loglevel", "info", "loglevel to enable")
	flag.Parse()

	var yggConfig *config.NodeConfig
	var popConfig *popura.PopuraConfig
	var err error
	switch {
	case *ver:
		fmt.Println("Build name:", version.BuildName())
		fmt.Println("Build version:", version.BuildVersion())
		return
	case *autoconf:
		// Use an autoconf-generated config, this will give us random keys and
		// port numbers, and will use an automatically selected TUN/TAP interface.
		yggConfig, popConfig = popura.GenerateConfig()
	case *useconffile != "" || *useconf:
		// Read the configuration from either stdin or from the filesystem
		yggConfig, popConfig = popura.LoadConfig(useconf, useconffile, normaliseconf)
		// If the -normaliseconf option was specified then remarshal the above
		// configuration and print it back to stdout. This lets the user update
		// their configuration file with newly mapped names (like above) or to
		// convert from plain JSON to commented HJSON.
		if *normaliseconf {
			fmt.Println(popura.SaveConfig(*yggConfig, *popConfig, *confjson))
			return
		}
	case *genconf:
		// Generate a new configuration and print it to stdout.
		yggConfig, popConfig = popura.GenerateConfig()
		fmt.Println(popura.SaveConfig(*yggConfig, *popConfig, *confjson))
		return
	case *meshnameconf != "":
		if conf, err := _meshname.GenConf(*meshnameconf, "meshname."); err == nil {
			fmt.Println(conf)
		} else {
			panic(err)
		}
		return
	default:
		// No flags were provided, therefore print the list of flags to stdout.
		flag.PrintDefaults()
	}
	// Have we got a working configuration? If we don't then it probably means
	// that neither -autoconf, -useconf or -useconffile were set above. Stop
	// if we don't.
	if yggConfig == nil {
		return
	}
	// Have we been asked for the node address yet? If so, print it and then stop.
	getNodeID := func() *crypto.NodeID {
		if pubkey, err := hex.DecodeString(yggConfig.EncryptionPublicKey); err == nil {
			var box crypto.BoxPubKey
			copy(box[:], pubkey[:])
			return crypto.GetNodeID(&box)
		}
		return nil
	}
	switch {
	case *getaddr:
		if nodeid := getNodeID(); nodeid != nil {
			addr := *address.AddrForNodeID(nodeid)
			ip := net.IP(addr[:])
			fmt.Println(ip.String())
		}
		return
	case *getsnet:
		if nodeid := getNodeID(); nodeid != nil {
			snet := *address.SubnetForNodeID(nodeid)
			ipnet := net.IPNet{
				IP:   append(snet[:], 0, 0, 0, 0, 0, 0, 0, 0),
				Mask: net.CIDRMask(len(snet)*8, 128),
			}
			fmt.Println(ipnet.String())
		}
		return
	default:
	}

	// Create a new logger that logs output to stdout.
	var logger *log.Logger
	switch *logto {
	case "stdout":
		logger = log.New(os.Stdout, "", log.Flags())
	case "syslog":
		if syslogger, err := gsyslog.NewLogger(gsyslog.LOG_NOTICE, "DAEMON", version.BuildName()); err == nil {
			logger = log.New(syslogger, "", log.Flags())
		}
	default:
		if logfd, err := os.OpenFile(*logto, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
			logger = log.New(logfd, "", log.Flags())
		}
	}
	if logger == nil {
		logger = log.New(os.Stdout, "", log.Flags())
		logger.Warnln("Logging defaulting to stdout")
	}

	setLogLevel(*loglevel, logger)

	// Setup the Yggdrasil node itself. The node{} type includes a Core, so we
	// don't need to create this manually.
	n := node{}
	// Now start Yggdrasil - this starts the DHT, router, switch and other core
	// components needed for Yggdrasil to operate
	n.state, err = n.core.Start(yggConfig, logger)
	if err != nil {
		logger.Errorln("An error occurred during startup")
		panic(err)
	}
	// Register the session firewall gatekeeper function
	n.core.SetSessionGatekeeper(n.sessionFirewall)
	// Allocate our modules
	n.admin = &admin.AdminSocket{}
	n.multicast = &multicast.Multicast{}
	n.tuntap = &tuntap.TunAdapter{}
	n.meshname = &meshname.MeshnameServer{}
	n.radv = &radv.RAdv{}
	n.autopeering = &autopeering.AutoPeering{}
	// Start the admin socket
	n.admin.Init(&n.core, n.state, logger, nil)
	if err := n.admin.Start(); err != nil {
		logger.Errorln("An error occurred starting admin socket:", err)
	}
	n.admin.SetupAdminHandlers(n.admin.(*admin.AdminSocket))
	// Start the multicast interface
	n.multicast.Init(&n.core, n.state, logger, nil)
	if err := n.multicast.Start(); err != nil {
		logger.Errorln("An error occurred starting multicast:", err)
	}
	n.multicast.SetupAdminHandlers(n.admin.(*admin.AdminSocket))
	// Start the TUN/TAP interface
	if listener, err := n.core.ConnListen(); err == nil {
		if dialer, err := n.core.ConnDialer(); err == nil {
			n.tuntap.Init(&n.core, n.state, logger, tuntap.TunOptions{Listener: listener, Dialer: dialer})
			if err := n.tuntap.Start(); err != nil {
				logger.Errorln("An error occurred starting TUN/TAP:", err)
			}
			n.tuntap.SetupAdminHandlers(n.admin.(*admin.AdminSocket))
		} else {
			logger.Errorln("Unable to get Dialer:", err)
		}
	} else {
		logger.Errorln("Unable to get Listener:", err)
	}
	// Start the DNS server
	n.meshname.Init(&n.core, n.state, popConfig, logger, nil)
	n.meshname.Start()

	// Start Router Advertisement module
	n.radv.Init(&n.core, n.state, popConfig, logger, nil)
	if err := n.radv.Start(); err != nil {
		logger.Errorln("An error occured starting RAdv: ", err)
	}

	n.autopeering.Init(&n.core, n.state, popConfig, logger, nil)

	// Make some nice output that tells us what our IPv6 address and subnet are.
	// This is just logged to stdout for the user.
	address := n.core.Address()
	subnet := n.core.Subnet()
	logger.Infof("Your IPv6 address is %s", address.String())
	logger.Infof("Your IPv6 subnet is %s", subnet.String())
	// Catch interrupts from the operating system to exit gracefully.
	c := make(chan os.Signal, 1)
	r := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	signal.Notify(r, os.Interrupt, syscall.SIGHUP)
	// Capture the service being stopped on Windows.
	minwinsvc.SetOnExit(n.shutdown)
	defer n.shutdown()

	// Setup auto peering
	if *autopeer && len(yggConfig.Peers) == 0 {
		n.autopeering.Start()
	}

	if *dhtcrawlenable {
		dhtcrawler := &dhtcrawler.Crawler{}
		dhtcrawler.Init(&n.core, logger)
		dhtcrawler.SetupAdminHandlers(n.admin.(*admin.AdminSocket))
	}

	// Wait for the terminate/interrupt signal. Once a signal is received, the
	// deferred Stop function above will run which will shut down TUN/TAP.
	for {
		select {
		case _ = <-c:
			goto exit
		case _ = <-r:
			if *useconffile != "" {
				yggConfig, popConfig = popura.LoadConfig(useconf, useconffile, normaliseconf)
				logger.Infoln("Reloading configuration from", *useconffile)
				n.core.UpdateConfig(yggConfig)
				n.tuntap.UpdateConfig(yggConfig)
				n.multicast.UpdateConfig(yggConfig)
				n.meshname.UpdateConfig(yggConfig, popConfig)
				n.radv.UpdateConfig(yggConfig, popConfig)
			} else {
				logger.Errorln("Reloading config at runtime is only possible with -useconffile")
			}
		}
	}
exit:
}

func (n *node) shutdown() {
	n.autopeering.Stop()
	n.radv.Stop()
	n.meshname.Stop()
	n.admin.Stop()
	n.multicast.Stop()
	n.tuntap.Stop()
	n.core.Stop()
}

func (n *node) sessionFirewall(pubkey *crypto.BoxPubKey, initiator bool) bool {
	n.state.Mutex.RLock()
	defer n.state.Mutex.RUnlock()

	// Allow by default if the session firewall is disabled
	if !n.state.Current.SessionFirewall.Enable {
		return true
	}

	// Prepare for checking whitelist/blacklist
	var box crypto.BoxPubKey
	// Reject blacklisted nodes
	for _, b := range n.state.Current.SessionFirewall.BlacklistEncryptionPublicKeys {
		key, err := hex.DecodeString(b)
		if err == nil {
			copy(box[:crypto.BoxPubKeyLen], key)
			if box == *pubkey {
				return false
			}
		}
	}

	// Allow whitelisted nodes
	for _, b := range n.state.Current.SessionFirewall.WhitelistEncryptionPublicKeys {
		key, err := hex.DecodeString(b)
		if err == nil {
			copy(box[:crypto.BoxPubKeyLen], key)
			if box == *pubkey {
				return true
			}
		}
	}

	// Allow outbound sessions if appropriate
	if n.state.Current.SessionFirewall.AlwaysAllowOutbound {
		if initiator {
			return true
		}
	}

	// Look and see if the pubkey is that of a direct peer
	var isDirectPeer bool
	for _, peer := range n.core.GetPeers() {
		if peer.PublicKey == *pubkey {
			isDirectPeer = true
			break
		}
	}

	// Allow direct peers if appropriate
	if n.state.Current.SessionFirewall.AllowFromDirect && isDirectPeer {
		return true
	}

	// Allow remote nodes if appropriate
	if n.state.Current.SessionFirewall.AllowFromRemote && !isDirectPeer {
		return true
	}

	// Finally, default-deny if not matching any of the above rules
	return false
}

func main() {
	run_yggdrasil()
}
