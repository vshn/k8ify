package internal

type Config struct {
	OutputDir   string   `json:"outputDir"`
	Env         string   `json:"env"`
	Ref         string   `json:"ref"`
	ConfigFiles []string `json:"configFiles"`
}
