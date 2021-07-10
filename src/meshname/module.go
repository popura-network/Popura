package meshname

import (
	"net"

	"github.com/gologme/log"

	"github.com/yggdrasil-network/yggdrasil-go/src/admin"
	"github.com/yggdrasil-network/yggdrasil-go/src/config"
	"github.com/yggdrasil-network/yggdrasil-go/src/core"

	_meshname "github.com/zhoreeq/meshname/pkg/meshname"

	"github.com/popura-network/Popura/src/popura"
)

type MeshnameServer struct {
	server *_meshname.MeshnameServer
	log    *log.Logger
	enable bool
}

func (s *MeshnameServer) Init(yggcore *core.Core, yggConfig *config.NodeConfig, popConfig *popura.PopuraConfig, log *log.Logger, options interface{}) error {
	s.log = log
	s.enable = popConfig.Meshname.Enable
	yggIPNet := &net.IPNet{IP: net.ParseIP("200::"), Mask: net.CIDRMask(7, 128)}
	s.server = _meshname.New(
		log,
		popConfig.Meshname.Listen,
		map[string]*net.IPNet{"ygg": yggIPNet, "meshname": yggIPNet, "vapordns": yggIPNet},
		false, // enable meship protocol
		false, // allow remote queries
	)

	if dnsRecords, err := _meshname.ParseDNSRecordsMap(popConfig.Meshname.Config); err == nil {
		s.server.ConfigureDNSRecords(dnsRecords)
	} else {
		s.log.Errorln("meshname: Failed to parse Meshname config:", err)
	}

	return nil
}

func (s *MeshnameServer) Start() error {
	if s.enable {
		return s.server.Start()
	} else {
		return nil
	}
}

func (s *MeshnameServer) Stop() error {
	s.server.Stop()
	return nil
}

func (s *MeshnameServer) UpdateConfig(yggConfig *config.NodeConfig, popConfig *popura.PopuraConfig) {
	// TODO Handle Enable/Disable and Listen
	if dnsRecords, err := _meshname.ParseDNSRecordsMap(popConfig.Meshname.Config); err == nil {
		s.server.ConfigureDNSRecords(dnsRecords)
	} else {
		s.log.Errorln("meshname: Failed to parse Meshname config:", err)
	}
}

func (s *MeshnameServer) SetupAdminHandlers(a *admin.AdminSocket) {}

func (s *MeshnameServer) IsStarted() bool {
	return s.server.IsStarted()
}
