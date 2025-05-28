package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config 应用配置
type Config struct {
	CanServiceURL string         `json:"can_service_url"`
	DefaultDevice DeviceConfig   `json:"default_device"`
	Devices       []DeviceConfig `json:"devices"`
	Server        ServerConfig   `json:"server"`
}

// DeviceConfig 设备配置
type DeviceConfig struct {
	ID           string                 `json:"id"`
	Model        string                 `json:"model"`
	CanInterface string                 `json:"can_interface"`
	Parameters   map[string]interface{} `json:"parameters"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port       int    `json:"port"`
	Host       string `json:"host"`
	LogLevel   string `json:"log_level"`
	EnableCORS bool   `json:"enable_cors"`
}

// LoadConfig 从文件加载配置
func LoadConfig(configPath string) (*Config, error) {
	file, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("打开配置文件失败：%w", err)
	}
	defer file.Close()

	var config Config
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败：%w", err)
	}

	// 设置默认值
	if config.Server.Port == 0 {
		config.Server.Port = 8080
	}
	if config.Server.Host == "" {
		config.Server.Host = "localhost"
	}
	if config.Server.LogLevel == "" {
		config.Server.LogLevel = "info"
	}

	return &config, nil
}

// SaveConfig 保存配置到文件
func SaveConfig(config *Config, configPath string) error {
	file, err := os.Create(configPath)
	if err != nil {
		return fmt.Errorf("创建配置文件失败：%w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(config); err != nil {
		return fmt.Errorf("保存配置文件失败：%w", err)
	}

	return nil
}

// GetDefaultConfig 获取默认配置
func GetDefaultConfig() *Config {
	return &Config{
		CanServiceURL: "http://localhost:8081",
		DefaultDevice: DeviceConfig{
			ID:           "left_hand",
			Model:        "L10",
			CanInterface: "can0",
			Parameters:   make(map[string]interface{}),
		},
		Devices: []DeviceConfig{
			{
				ID:           "left_hand",
				Model:        "L10",
				CanInterface: "can0",
				Parameters: map[string]interface{}{
					"hand_type": "left",
				},
			},
		},
		Server: ServerConfig{
			Port:       8080,
			Host:       "localhost",
			LogLevel:   "info",
			EnableCORS: true,
		},
	}
}
