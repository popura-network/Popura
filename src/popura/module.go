package popura

import (
	"github.com/gologme/log"

	"github.com/yggdrasil-network/yggdrasil-go/src/admin"
	"github.com/yggdrasil-network/yggdrasil-go/src/config"
	"github.com/yggdrasil-network/yggdrasil-go/src/core"
)

// Module is an interface that defines which functions must be supported by a
// given Popura module.
type Module interface {
	Init(yggcore *core.Core, yggConfig *config.NodeConfig, popuraConf *PopuraConfig, log *log.Logger, options interface{}) error
	Start() error
	Stop() error
	UpdateConfig(yggConf *config.NodeConfig, popuraConf *PopuraConfig)
	SetupAdminHandlers(a *admin.AdminSocket)
	IsStarted() bool
}
