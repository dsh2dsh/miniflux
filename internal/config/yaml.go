package config

type yamlOptions struct {
	HostLimits   map[string]HostLimits `yaml:"host_limits" validate:"dive,keys,required,endkeys,required"`
	PrivateHosts map[string][]string   `yaml:"privateHosts" validate:"dive,keys,required,ip|hostname_port,endkeys,dive,required,url"`
}

type HostLimits struct {
	Connections int64   `yaml:"connections" validate:"omitempty,min=0"`
	Rate        float64 `yaml:"rate" validate:"omitempty,min=0"`
}
