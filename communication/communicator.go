package communication

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"hands/config"
	"hands/define"
	"io"
	"net/http"
	"time"
)

// TODO: ID 的作用是什么
// RawMessage 代表发送给 can-bridge 服务或从其接收的原始消息结构
type RawMessage struct {
	Interface string `json:"interface"` // 目标 CAN 接口名，例如 "can0", "vcan1"
	ID        uint32 `json:"id"`        // CAN 帧的 ID
	Data      []byte `json:"data"`      // CAN 帧的数据负载
}

// Communicator 定义了与 can-bridge Web 服务进行通信的接口
type Communicator interface {
	// SendMessage 将 RawMessage 通过 HTTP POST 请求发送到 can-bridge 服务
	SendMessage(ctx context.Context, msg RawMessage) error

	// GetAllInterfaceStatuses 获取所有已知 CAN 接口的状态
	GetAllInterfaceStatuses() (statuses map[string]bool, err error)

	// SetServiceURL 设置 can-bridge 服务的 URL
	SetServiceURL(url string)

	// IsConnected 检查与 can-bridge 服务的连接状态
	IsConnected() bool
}

// CanBridgeClient 实现与 can-bridge 服务的 HTTP 通信
type CanBridgeClient struct {
	serviceURL string
	client     *http.Client
}

func NewCanBridgeClient(serviceURL string) Communicator {
	return &CanBridgeClient{
		serviceURL: serviceURL,
		client:     &http.Client{Timeout: 5 * time.Second},
	}
}

func (c *CanBridgeClient) SendMessage(ctx context.Context, msg RawMessage) error {
	jsonData, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("序列化消息失败：%w", err)
	}

	url := fmt.Sprintf("%s/api/can", c.serviceURL)

	// 创建带有 context 的请求
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("创建 HTTP 请求失败：%w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("发送 HTTP 请求失败：%w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("can-bridge服务返回错误: %d, %s", resp.StatusCode, string(body))
	}

	return nil
}

func (c *CanBridgeClient) GetAllInterfaceStatuses() (map[string]bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	url := fmt.Sprintf("%s/api/status", c.serviceURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("获取所有接口状态失败：%w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送 HTTP 请求失败：%w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("can-bridge 服务返回错误：%d", resp.StatusCode)
	}

	var statusResp define.ApiResponse
	if err := json.NewDecoder(resp.Body).Decode(&statusResp); err != nil {
		return nil, fmt.Errorf("解析状态响应失败：%w", err)
	}

	result := make(map[string]bool)
	for _, ifName := range config.Config.AvailableInterfaces {
		result[ifName] = false
	}

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

	return result, nil
}

func (c *CanBridgeClient) SetServiceURL(url string) { c.serviceURL = url }

func (c *CanBridgeClient) IsConnected() bool {
	_, err := c.GetAllInterfaceStatuses()
	return err == nil
}
