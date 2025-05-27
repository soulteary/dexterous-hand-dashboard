package hands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"hands/config"
	"hands/define"
	"log"
	"net/http"
)

type CanMessage struct {
	Interface string `json:"interface"`
	ID        uint32 `json:"id"`
	Data      []byte `json:"data"`
}

// 检查 CAN 服务状态
func CheckCanServiceStatus() map[string]bool {
	resp, err := http.Get(config.Config.CanServiceURL + "/api/status")
	if err != nil {
		log.Printf("❌ CAN 服务状态检查失败: %v", err)
		result := make(map[string]bool)
		for _, ifName := range config.Config.AvailableInterfaces {
			result[ifName] = false
		}
		return result
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("❌ CAN 服务返回非正常状态：%d", resp.StatusCode)
		result := make(map[string]bool)
		for _, ifName := range config.Config.AvailableInterfaces {
			result[ifName] = false
		}
		return result
	}

	var statusResp define.ApiResponse
	if err := json.NewDecoder(resp.Body).Decode(&statusResp); err != nil {
		log.Printf("❌ 解析 CAN 服务状态失败: %v", err)
		result := make(map[string]bool)
		for _, ifName := range config.Config.AvailableInterfaces {
			result[ifName] = false
		}
		return result
	}

	// 检查状态数据
	result := make(map[string]bool)
	for _, ifName := range config.Config.AvailableInterfaces {
		result[ifName] = false
	}

	// 从响应中获取各接口状态
	if statusData, ok := statusResp.Data.(map[string]interface{}); ok {
		if interfaces, ok := statusData["interfaces"].(map[string]interface{}); ok {
			for ifName, ifStatus := range interfaces {
				if status, ok := ifStatus.(map[string]interface{}); ok {
					if active, ok := status["active"].(bool); ok {
						result[ifName] = active
					}
				}
			}
		}
	}

	return result
}

// 发送请求到 CAN 服务
func sendToCanService(msg CanMessage) error {
	jsonData, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("JSON 编码错误: %v", err)
	}

	resp, err := http.Post(config.Config.CanServiceURL+"/api/can", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("CAN 服务请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp define.ApiResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			return fmt.Errorf("CAN 服务返回错误：HTTP %d", resp.StatusCode)
		}
		return fmt.Errorf("CAN 服务返回错误: %s", errResp.Error)
	}

	return nil
}
