package popura

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"os"

	"golang.org/x/text/encoding/unicode"

	"github.com/hjson/hjson-go"
	"github.com/mitchellh/mapstructure"

	"github.com/yggdrasil-network/yggdrasil-go/src/config"
)

type PopuraConfig struct {
	Meshname MeshnameConfig `comment:"DNS server description"`
	RAdv     RAdvConfig     `comment:"Router Advertisement settings"`
}

type MeshnameConfig struct {
	Enable bool                `comment:"Enable or disable the DNS server"`
	Listen string              `comment:"Listen address for the DNS server"`
	Config map[string][]string `comment:"DNS zone configuration"`
}

type RAdvConfig struct {
	Enable       bool   `comment:"Enable or disable Router Advertisement"`
	Interface    string `comment:"Send router advertisement for this network interface"`
	SetGatewayIP bool   `comment:"Set IP address on the Interface automatically"`
	DefaultRouter       bool   `comment:"Advertise a default router (a fix for Android)"`
	DNS bool   `comment:"Advertise this router as a DNS server (RFC 8106)"`
}

func GenerateConfig() (*config.NodeConfig, *PopuraConfig) {
	popConfig := PopuraConfig{}

	popConfig.Meshname.Enable = false
	popConfig.Meshname.Listen = "[::1]:53535"
	popConfig.Meshname.Config = map[string][]string{}

	popConfig.RAdv.Enable = false
	popConfig.RAdv.Interface = "eth0"
	popConfig.RAdv.SetGatewayIP = true
	popConfig.RAdv.DefaultRouter = false
	popConfig.RAdv.DNS = false

	return config.GenerateConfig(), &popConfig
}

// initialize empty values for correct JSON serialization
func correctEmptyValues(yggConfig *config.NodeConfig) {
	if len(yggConfig.TunnelRouting.IPv4LocalSubnets) == 0 {
		yggConfig.TunnelRouting.IPv4LocalSubnets = []string{}
	}
	if len(yggConfig.TunnelRouting.IPv6LocalSubnets) == 0 {
		yggConfig.TunnelRouting.IPv6LocalSubnets = []string{}
	}
	if len(yggConfig.TunnelRouting.IPv4RemoteSubnets) == 0 {
		yggConfig.TunnelRouting.IPv4RemoteSubnets = make(map[string]string)
	}
	if len(yggConfig.TunnelRouting.IPv6RemoteSubnets) == 0 {
		yggConfig.TunnelRouting.IPv6RemoteSubnets = make(map[string]string)
	}
	if len(yggConfig.SessionFirewall.WhitelistEncryptionPublicKeys) == 0 {
		yggConfig.SessionFirewall.WhitelistEncryptionPublicKeys = []string{}
	}
	if len(yggConfig.SessionFirewall.BlacklistEncryptionPublicKeys) == 0 {
		yggConfig.SessionFirewall.BlacklistEncryptionPublicKeys = []string{}
	}
	if len(yggConfig.NodeInfo) == 0 {
		yggConfig.NodeInfo = make(map[string]interface{})
	}
}

func SaveConfig(yggConfig config.NodeConfig, popConfig PopuraConfig, isjson bool) string {
	// combine config structs into one and marshal it
	// FIXME hjson comments are lost
	var combo map[string]interface{}

	correctEmptyValues(&yggConfig)
	ybs, _ := json.Marshal(&yggConfig)
	pbs, _ := json.Marshal(&popConfig)
	json.Unmarshal(ybs, &combo)
	json.Unmarshal(pbs, &combo)

	var res []byte
	var err error
	if isjson {
		res, err = json.MarshalIndent(combo, "", "  ")
	} else {
		res, err = hjson.Marshal(combo)
	}
	if err != nil {
		panic(err)
	}
	return string(res)
}

func LoadConfig(useconf *bool, useconffile *string, normaliseconf *bool) (*config.NodeConfig, *PopuraConfig) {
	var conf []byte
	var err error
	if *useconffile != "" {
		// Read the file from the filesystem
		conf, err = ioutil.ReadFile(*useconffile)
	} else {
		// Read the file from stdin.
		conf, err = ioutil.ReadAll(os.Stdin)
	}
	if err != nil {
		panic(err)
	}
	// If there's a byte order mark - which Windows 10 is now incredibly fond of
	// throwing everywhere when it's converting things into UTF-16 for the hell
	// of it - remove it and decode back down into UTF-8. This is necessary
	// because hjson doesn't know what to do with UTF-16 and will panic
	if bytes.Equal(conf[0:2], []byte{0xFF, 0xFE}) ||
		bytes.Equal(conf[0:2], []byte{0xFE, 0xFF}) {
		utf := unicode.UTF16(unicode.BigEndian, unicode.UseBOM)
		decoder := utf.NewDecoder()
		conf, err = decoder.Bytes(conf)
		if err != nil {
			panic(err)
		}
	}
	// Generate a new configuration - this gives us a set of sane defaults -
	// then parse the configuration we loaded above on top of it. The effect
	// of this is that any configuration item that is missing from the provided
	// configuration will use a sane default.
	yggConfig, popConfig := GenerateConfig()
	var dat map[string]interface{}
	if err := hjson.Unmarshal(conf, &dat); err != nil {
		panic(err)
	}

	// Sanitise the config
	confJson, err := json.Marshal(dat)
	if err != nil {
		panic(err)
	}
	json.Unmarshal(confJson, &yggConfig)
	json.Unmarshal(confJson, &popConfig)
	// Overlay our newly mapped configuration onto the autoconf node config that
	// we generated above.
	if err = mapstructure.Decode(dat, &yggConfig); err != nil {
		panic(err)
	}
	if err = mapstructure.Decode(dat, &popConfig); err != nil {
		panic(err)
	}
	return yggConfig, popConfig
}
