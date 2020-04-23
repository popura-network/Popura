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
	Yggdrasil     *config.NodeConfig
	TestParameter bool `comment:"Test parameter for Popura"`
}

func GenerateConfig() *PopuraConfig {
	pcfg := PopuraConfig{}
	pcfg.TestParameter = false

	pcfg.Yggdrasil = config.GenerateConfig()

	return &pcfg
}

func GenerateConfigString(isjson bool) string {
	cfg := GenerateConfig()
	var bs []byte
	var err error
	if isjson {
		bs, err = json.MarshalIndent(cfg, "", "  ")
	} else {
		bs, err = hjson.Marshal(cfg)
	}
	if err != nil {
		panic(err)
	}
	return string(bs)
}

func LoadConfig(useconf *bool, useconffile *string, normaliseconf *bool) *PopuraConfig {
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
	if bytes.Compare(conf[0:2], []byte{0xFF, 0xFE}) == 0 ||
		bytes.Compare(conf[0:2], []byte{0xFE, 0xFF}) == 0 {
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
	pcfg := GenerateConfig()
	var dat map[string]interface{}
	if err := hjson.Unmarshal(conf, &dat); err != nil {
		panic(err)
	}

	// Sanitise the config
	confJson, err := json.Marshal(dat)
	if err != nil {
		panic(err)
	}
	json.Unmarshal(confJson, &pcfg)
	// Overlay our newly mapped configuration onto the autoconf node config that
	// we generated above.
	if err = mapstructure.Decode(dat, &pcfg); err != nil {
		panic(err)
	}
	return pcfg
}
