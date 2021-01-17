package config

type Global struct {
	ExternalLabels map[string]string `json:"external_labels" yaml:"external_labels"`
}

type Prometheus struct {
	Output string `json:"output" yaml:"output"`
}

type Cluster struct {
	Peers         []string `json:"peers" yaml:"peers"`
	AdvertiseAddr string   `json:"advertise_addr" yaml:"advertise_addr"`
}

type Tasks struct {
	DryRun bool   `json:"dry_run" yaml:"dry_run"`
	States string `json:"states" yaml:"states"`
}

type Config struct {
	Global     Global     `json:"global" yaml:"global"`
	Prometheus Prometheus `json:"prometheus" yaml:"prometheus"`
	Cluster    Cluster    `json:"cluster" yaml:"cluster"`
	Tasks      Tasks      `json:"tasks" yaml:"tasks"`
}

func (config *Config) Valid() error {
	return nil
}
