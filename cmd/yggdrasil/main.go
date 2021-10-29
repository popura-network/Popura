package main

import (
	"context"
	"crypto/ed25519"
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

	"github.com/yggdrasil-network/yggdrasil-go/src/core"
	"github.com/yggdrasil-network/yggdrasil-go/src/multicast"
	"github.com/yggdrasil-network/yggdrasil-go/src/tuntap"
	"github.com/yggdrasil-network/yggdrasil-go/src/version"

	_meshname "github.com/zhoreeq/meshname/pkg/meshname"

	"github.com/popura-network/Popura/src/autopeering"
	"github.com/popura-network/Popura/src/meshname"
	"github.com/popura-network/Popura/src/popura"
)

type node struct {
	core        core.Core
	config      *config.NodeConfig
	tuntap      *tuntap.TunAdapter
	multicast   *multicast.Multicast
	admin       *admin.AdminSocket
	meshname    popura.Module // meshname.MeshnameServer
	autopeering popura.Module // autopeering.AutoPeering
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

type yggArgs struct {
	genconf       bool
	useconf       bool
	useconffile   string
	normaliseconf bool
	confjson      bool
	autoconf      bool
	ver           bool
	logto         string
	getaddr       bool
	getsnet       bool
	loglevel      string
	autopeer      bool
	meshnameconf  bool
	withpeers     int
}

func getArgs() yggArgs {
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
	meshnameconf := flag.Bool("meshnameconf", false, "use with -useconffile. Prints config with a default meshname DNS record")
	withpeers := flag.Int("withpeers", 0, "generate a config with N number of alive peers")
	loglevel := flag.String("loglevel", "info", "loglevel to enable")
	flag.Parse()
	return yggArgs{
		genconf:       *genconf,
		useconf:       *useconf,
		useconffile:   *useconffile,
		normaliseconf: *normaliseconf,
		confjson:      *confjson,
		autoconf:      *autoconf,
		autopeer:      *autopeer,
		ver:           *ver,
		logto:         *logto,
		getaddr:       *getaddr,
		getsnet:       *getsnet,
		meshnameconf:  *meshnameconf,
		withpeers:     *withpeers,
		loglevel:      *loglevel,
	}
}

// The main function is responsible for configuring and starting Yggdrasil.
func run(args yggArgs, ctx context.Context, done chan struct{}) {
	defer close(done)
	// Create a new logger that logs output to stdout.
	var logger *log.Logger
	switch args.logto {
	case "stdout":
		logger = log.New(os.Stdout, "", log.Flags())
	case "syslog":
		if syslogger, err := gsyslog.NewLogger(gsyslog.LOG_NOTICE, "DAEMON", version.BuildName()); err == nil {
			logger = log.New(syslogger, "", log.Flags())
		}
	default:
		if logfd, err := os.OpenFile(args.logto, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
			logger = log.New(logfd, "", log.Flags())
		}
	}
	if logger == nil {
		logger = log.New(os.Stdout, "", log.Flags())
		logger.Warnln("Logging defaulting to stdout")
	}

	if args.normaliseconf {
		setLogLevel("error", logger)
	} else {
		setLogLevel(args.loglevel, logger)
	}

	var yggConfig *config.NodeConfig
	var popConfig *popura.PopuraConfig
	var err error
	switch {
	case args.ver:
		fmt.Println("Build name:", version.BuildName())
		fmt.Println("Build version:", version.BuildVersion())
		return
	case args.autoconf:
		// Use an autoconf-generated config, this will give us random keys and
		// port numbers, and will use an automatically selected TUN/TAP interface.
		yggConfig, popConfig = popura.GenerateConfig()
	case args.useconffile != "" || args.useconf:
		// Read the configuration from either stdin or from the filesystem
		yggConfig, popConfig = popura.LoadConfig(&args.useconf, &args.useconffile, &args.normaliseconf)
		// If the -normaliseconf option was specified then remarshal the above
		// configuration and print it back to stdout. This lets the user update
		// their configuration file with newly mapped names (like above) or to
		// convert from plain JSON to commented HJSON.
		if args.normaliseconf {
			fmt.Println(popura.SaveConfig(*yggConfig, *popConfig, args.confjson))
			return
		}

		if args.meshnameconf {
			ip := popura.AddressFromKey(yggConfig.PrivateKey)
			subDomain := _meshname.DomainFromIP(ip)

			defaultRecord := fmt.Sprintf("%s.vapordns AAAA %s", subDomain, ip.String())
			meshnameConfig := make(map[string][]string)
			meshnameConfig[subDomain] = []string{defaultRecord}

			popConfig.Meshname.Enable = true
			popConfig.Meshname.Config = meshnameConfig
			fmt.Println(popura.SaveConfig(*yggConfig, *popConfig, args.confjson))
			return
		}
	case args.genconf:
		// Generate a new configuration and print it to stdout.
		yggConfig, popConfig = popura.GenerateConfig()
		if args.withpeers > 0 {
			apeers := autopeering.RandomPick(autopeering.GetClosestPeers(autopeering.GetPublicPeers(), 10), args.withpeers)
			for _, p := range apeers {
				yggConfig.Peers = append(yggConfig.Peers, p.String())
			}
		}
		fmt.Println(popura.SaveConfig(*yggConfig, *popConfig, args.confjson))
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

	// i2p:// and onion:// peer URI support
	i2pSocks := "socks://127.0.0.1:4447/"
	onionSocks := "socks://127.0.0.1:9050/"

	if os.Getenv("I2P_SOCKS") != "" {
		i2pSocks = os.Getenv("I2P_SOCKS")
	}
	if os.Getenv("ONION_SOCKS") != "" {
		onionSocks = os.Getenv("ONION_SOCKS")
	}

	for i, peer := range yggConfig.Peers {
		if strings.HasPrefix(peer, "i2p:") {
			yggConfig.Peers[i] = i2pSocks + peer[6:]
		} else if strings.HasPrefix(peer, "onion:") {
			yggConfig.Peers[i] =  onionSocks + peer[8:]
		}
	}

	// Have we been asked for the node address yet? If so, print it and then stop.
	getNodeKey := func() ed25519.PublicKey {
		if pubkey, err := hex.DecodeString(yggConfig.PrivateKey); err == nil {
			return ed25519.PrivateKey(pubkey).Public().(ed25519.PublicKey)
		}
		return nil
	}
	switch {
	case args.getaddr:
		if key := getNodeKey(); key != nil {
			addr := address.AddrForKey(key)
			ip := net.IP(addr[:])
			fmt.Println(ip.String())
		}
		return
	case args.getsnet:
		if key := getNodeKey(); key != nil {
			snet := address.SubnetForKey(key)
			ipnet := net.IPNet{
				IP:   append(snet[:], 0, 0, 0, 0, 0, 0, 0, 0),
				Mask: net.CIDRMask(len(snet)*8, 128),
			}
			fmt.Println(ipnet.String())
		}
		return
	default:
	}

	// Setup the Yggdrasil node itself. The node{} type includes a Core, so we
	// don't need to create this manually.
	n := node{config: yggConfig}
	// Now start Yggdrasil - this starts the DHT, router, switch and other core
	// components needed for Yggdrasil to operate
	if err = n.core.Start(yggConfig, logger); err != nil {
		logger.Errorln("An error occurred during startup")
		panic(err)
	}
	// Register the session firewall gatekeeper function
	// Allocate our modules
	n.admin = &admin.AdminSocket{}
	n.multicast = &multicast.Multicast{}
	n.tuntap = &tuntap.TunAdapter{}
	n.meshname = &meshname.MeshnameServer{}
	n.autopeering = &autopeering.AutoPeering{}
	// Start the admin socket
	if err := n.admin.Init(&n.core, yggConfig, logger, nil); err != nil {
		logger.Errorln("An error occurred initialising admin socket:", err)
	} else if err := n.admin.Start(); err != nil {
		logger.Errorln("An error occurred starting admin socket:", err)
	}
	n.admin.SetupAdminHandlers(n.admin)
	// Start the multicast interface
	if err := n.multicast.Init(&n.core, yggConfig, logger, nil); err != nil {
		logger.Errorln("An error occurred initialising multicast:", err)
	} else if err := n.multicast.Start(); err != nil {
		logger.Errorln("An error occurred starting multicast:", err)
	}
	n.multicast.SetupAdminHandlers(n.admin)
	// Start the TUN/TAP interface
	if err := n.tuntap.Init(&n.core, yggConfig, logger, nil); err != nil {
		logger.Errorln("An error occurred initialising TUN/TAP:", err)
	} else if err := n.tuntap.Start(); err != nil {
		logger.Errorln("An error occurred starting TUN/TAP:", err)
	}
	n.tuntap.SetupAdminHandlers(n.admin)
	// Start the DNS server
	n.meshname.Init(&n.core, yggConfig, popConfig, logger, nil)
	n.meshname.Start()

	n.autopeering.Init(&n.core, yggConfig, popConfig, logger, nil)
	// Setup auto peering
	if args.autopeer && len(yggConfig.Peers) == 0 {
		n.autopeering.Start()
	}

	// Make some nice output that tells us what our IPv6 address and subnet are.
	// This is just logged to stdout for the user.
	address := n.core.Address()
	subnet := n.core.Subnet()
	public := n.core.GetSelf().Key
	logger.Infof("Your public key is %s", hex.EncodeToString(public[:]))
	logger.Infof("Your IPv6 address is %s", address.String())
	logger.Infof("Your IPv6 subnet is %s", subnet.String())
	// Catch interrupts from the operating system to exit gracefully.
	<-ctx.Done()
	// Capture the service being stopped on Windows.
	minwinsvc.SetOnExit(n.shutdown)
	n.shutdown()
}

func (n *node) shutdown() {
	_ = n.autopeering.Stop()
	_ = n.meshname.Stop()
	_ = n.admin.Stop()
	_ = n.multicast.Stop()
	_ = n.tuntap.Stop()
	n.core.Stop()
}

func main() {
	args := getArgs()
	hup := make(chan os.Signal, 1)
	//signal.Notify(hup, os.Interrupt, syscall.SIGHUP)
	term := make(chan os.Signal, 1)
	signal.Notify(term, os.Interrupt, syscall.SIGTERM)
	for {
		done := make(chan struct{})
		ctx, cancel := context.WithCancel(context.Background())
		go run(args, ctx, done)
		select {
		case <-hup:
			cancel()
			<-done
		case <-term:
			cancel()
			<-done
			return
		case <-done:
			return
		}
	}
}
