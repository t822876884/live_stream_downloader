package config

// Config 应用配置
type Config struct {
	ServerAddr string // 服务器地址
	DataDir    string // 数据目录
}

// NewDefaultConfig 创建默认配置
func NewDefaultConfig() *Config {
	return &Config{
		ServerAddr: ":8080",
		DataDir:    "./data",
	}
}