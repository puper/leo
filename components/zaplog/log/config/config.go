package config

type LogConfig struct {
	Level         string `json:"level"`
	TraceLevel    string `json:"traceLevel"`
	Output        string `json:"output"`
	MaxSize       int    `json:"maxSize"`
	MaxAge        int    `json:"maxAge"`
	MaxBackups    int    `json:"maxBackups"`
	Compress      bool   `json:"compress"`
	InitialFields []any  `json:"initialFields"`
	Format        string `json:"format"`
}

type Config struct {
	Logs map[string]*LogConfig `json:"logs"`
}
