package cli

import (
	"encoding/json"
	"flag"
	"hands/define"
	"log"
	"net/http"
	"os"
	"strings"
)

// 解析配置
func ParseConfig() *define.Config {
	cfg := &define.Config{}

	// 命令行参数
	var canInterfacesFlag string
	flag.StringVar(&cfg.CanServiceURL, "can-url", "http://127.0.0.1:5260", "CAN 服务的 URL")
	flag.StringVar(&cfg.WebPort, "port", "9099", "Web 服务的端口")
	flag.StringVar(&cfg.DefaultInterface, "interface", "", "默认 CAN 接口")
	flag.StringVar(&canInterfacesFlag, "can-interfaces", "", "支持的 CAN 接口列表，用逗号分隔 (例如：can0,can1,vcan0)")
	flag.Parse()

	// 环境变量覆盖命令行参数
	if envURL := os.Getenv("CAN_SERVICE_URL"); envURL != "" {
		cfg.CanServiceURL = envURL
	}
	if envPort := os.Getenv("WEB_PORT"); envPort != "" {
		cfg.WebPort = envPort
	}
	if envInterface := os.Getenv("DEFAULT_INTERFACE"); envInterface != "" {
		cfg.DefaultInterface = envInterface
	}
	if envInterfaces := os.Getenv("CAN_INTERFACES"); envInterfaces != "" {
		canInterfacesFlag = envInterfaces
	}

	// 解析可用接口
	if canInterfacesFlag != "" {
		cfg.AvailableInterfaces = strings.Split(canInterfacesFlag, ",")
		// 清理空白字符
		for i, iface := range cfg.AvailableInterfaces {
			cfg.AvailableInterfaces[i] = strings.TrimSpace(iface)
		}
	}

	// 如果没有指定可用接口，从 CAN 服务获取
	if len(cfg.AvailableInterfaces) == 0 {
		log.Println("🔍 未指定可用接口，将从 CAN 服务获取...")
		cfg.AvailableInterfaces = getAvailableInterfacesFromCanService(cfg.CanServiceURL)
	}

	// 设置默认接口
	if cfg.DefaultInterface == "" && len(cfg.AvailableInterfaces) > 0 {
		cfg.DefaultInterface = cfg.AvailableInterfaces[0]
	}

	return cfg
}

// 从 CAN 服务获取可用接口
func getAvailableInterfacesFromCanService(canServiceURL string) []string {
	resp, err := http.Get(canServiceURL + "/api/interfaces")
	if err != nil {
		log.Printf("⚠️ 无法从 CAN 服务获取接口列表: %v，使用默认配置", err)
		return []string{"can0", "can1"} // 默认接口
	}
	defer resp.Body.Close()

	var apiResp define.ApiResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		log.Printf("⚠️ 解析 CAN 服务接口响应失败: %v，使用默认配置", err)
		return []string{"can0", "can1"}
	}

	if data, ok := apiResp.Data.(map[string]interface{}); ok {
		if configuredPorts, ok := data["configuredPorts"].([]interface{}); ok {
			interfaces := make([]string, 0, len(configuredPorts))
			for _, port := range configuredPorts {
				if portStr, ok := port.(string); ok {
					interfaces = append(interfaces, portStr)
				}
			}
			if len(interfaces) > 0 {
				log.Printf("✅ 从 CAN 服务获取到接口: %v", interfaces)
				return interfaces
			}
		}
	}

	log.Println("⚠️ 无法从 CAN 服务获取有效接口，使用默认配置")
	return []string{"can0", "can1"}
}
