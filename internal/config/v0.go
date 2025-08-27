package config

const configVersionV0 = "0"

type configV0 struct {
	Version string `json:"version"` // required by vconfig-go
}

// newConfigV0 creates a new v0 configuration
func newConfigV0() *configV0 {
	return &configV0{
		Version: configVersionV0,
	}
}

func (c *configV0) validateV0() error {
	return nil
}
