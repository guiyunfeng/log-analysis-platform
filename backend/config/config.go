package config

import (
	"log"
	"os"
)

type Config struct {
	MySQLDSN         string
	LokiURL          string
	DingTalkWebhook  string
	ServerPort       string
}

var GlobalConfig Config

func Load() {
	GlobalConfig = Config{
		MySQLDSN:        getEnv("MYSQL_DSN", "root:password@tcp(127.0.0.1:3306)/log_analysis?charset=utf8mb4&parseTime=True&loc=Local"),
		LokiURL:         getEnv("LOKI_URL", "http://localhost:3100"),
		DingTalkWebhook: getEnv("DINGTALK_WEBHOOK", ""),
		ServerPort:      getEnv("SERVER_PORT", "8080"),
	}
	log.Printf("Config loaded: LokiURL=%s, ServerPort=%s", GlobalConfig.LokiURL, GlobalConfig.ServerPort)
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
