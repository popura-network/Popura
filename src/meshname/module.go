package meshname

import (
	"net"

	"github.com/gologme/log"

	"github.com/yggdrasil-network/yggdrasil-go/src/admin"
	"github.com/yggdrasil-network/yggdrasil-go/src/config"
	"github.com/yggdrasil-network/yggdrasil-go/src/yggdrasil"

	_meshname "github.com/zhoreeq/meshname/src/meshname"

	"github.com/popura-network/Popura/src/popura"
)

type MeshnameServer struct {
	core        *yggdrasil.Core
	server	*_meshname.MeshnameServer
	log         *log.Logger
	enable bool
}

func (s *MeshnameServer) Init(core *yggdrasil.Core, state *config.NodeState, popConfig *popura.PopuraConfig, log *log.Logger, options interface{}) error {
	s.log = log
	s.server = &_meshname.MeshnameServer{}
	s.server.Init(log, popConfig.Meshname.Listen)
	yggIPNet := &net.IPNet{net.ParseIP("200::"), net.CIDRMask(7, 128)}
	s.enable = popConfig.Meshname.Enable
	s.server.SetNetworks(map[string]*net.IPNet{"ygg": yggIPNet, "meshname": yggIPNet})
	if zoneConfig, err := _meshname.ParseZoneConfigMap(popConfig.Meshname.Config); err == nil {
		s.server.SetZoneConfig(zoneConfig)
	} else {
		s.log.Errorln("meshname: Failed to parse Meshname config:", err)
	}

	return nil
}
func (s *MeshnameServer) Start() error {
	if s.enable == true {
		return s.server.Start()
	} else {
		return nil
	}
}

func (s *MeshnameServer) Stop() error {
	return s.server.Stop()
}

func (s *MeshnameServer) UpdateConfig(yggConfig *config.NodeConfig, popConfig *popura.PopuraConfig) {
	// TODO Handle Enable/Disable and Listen
	if zoneConfig, err := _meshname.ParseZoneConfigMap(popConfig.Meshname.Config); err == nil {
		s.server.SetZoneConfig(zoneConfig)
	} else {
		s.log.Errorln("meshname: Failed to parse Meshname config:", err)
	}
}

func (s *MeshnameServer) SetupAdminHandlers(a *admin.AdminSocket) {}

func (s *MeshnameServer) IsStarted() bool {
	return s.server.IsStarted()
}
