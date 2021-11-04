package popura

type PopuraConfig struct {
	Autopeering AutopeeringConfig `comment:"Autopeering description"`
	Meshname    MeshnameConfig    `comment:"DNS server description"`
}

type AutopeeringConfig struct {
	Enable bool `comment:"Enable autopeering"`
}

type MeshnameConfig struct {
	Enable bool   `comment:"Enable or disable the DNS server"`
	Listen string `comment:"Listen address for the DNS server"`
}

func GenerateConfig() *PopuraConfig {
	popConfig := PopuraConfig{}

	popConfig.Autopeering.Enable = false

	popConfig.Meshname.Enable = false
	popConfig.Meshname.Listen = "[::1]:53535"

	return &popConfig
}
