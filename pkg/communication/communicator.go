package communication

import (
	"bytes"
	"encoding/json"
	"fmt"
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
	SendMessage(msg RawMessage) error

	// GetInterfaceStatus 获取指定 CAN 接口的状态
	GetInterfaceStatus(ifName string) (isActive bool, err error)

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
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (c *CanBridgeClient) SendMessage(msg RawMessage) error {
	jsonData, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("序列化消息失败：%w", err)
	}

	url := fmt.Sprintf("%s/api/can", c.serviceURL)
	resp, err := c.client.Post(url, "application/json", bytes.NewBuffer(jsonData))
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

func (c *CanBridgeClient) GetInterfaceStatus(ifName string) (bool, error) {
	url := fmt.Sprintf("%s/api/status/%s", c.serviceURL, ifName)
	resp, err := c.client.Get(url)
	if err != nil {
		return false, fmt.Errorf("获取接口状态失败：%w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("can-bridge 服务返回错误：%d", resp.StatusCode)
	}

	var status struct {
		Active bool `json:"active"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return false, fmt.Errorf("解析状态响应失败：%w", err)
	}

	return status.Active, nil
}

func (c *CanBridgeClient) GetAllInterfaceStatuses() (map[string]bool, error) {
	url := fmt.Sprintf("%s/api/status", c.serviceURL)
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("获取所有接口状态失败：%w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("can-bridge 服务返回错误：%d", resp.StatusCode)
	}

	var statuses map[string]bool
	if err := json.NewDecoder(resp.Body).Decode(&statuses); err != nil {
		return nil, fmt.Errorf("解析状态响应失败：%w", err)
	}

	return statuses, nil
}

func (c *CanBridgeClient) SetServiceURL(url string) { c.serviceURL = url }

func (c *CanBridgeClient) IsConnected() bool {
	_, err := c.GetAllInterfaceStatuses()
	return err == nil
}
