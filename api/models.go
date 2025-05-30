package api

import (
	"hands/device"
	"time"
)

// ===== 通用响应模型 =====

// ApiResponse 统一 API 响应格式（保持与原 API 兼容）
type ApiResponse struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
	Data    any    `json:"data,omitempty"`
}

// ===== 设备管理相关模型 =====

// DeviceCreateRequest 创建设备请求
type DeviceCreateRequest struct {
	ID       string         `json:"id" binding:"required"`
	Model    string         `json:"model" binding:"required"`
	Config   map[string]any `json:"config"`
	HandType string         `json:"handType,omitempty"` // "left" 或 "right"
}

// DeviceInfo 设备信息响应
type DeviceInfo struct {
	ID       string              `json:"id"`
	Model    string              `json:"model"`
	HandType string              `json:"handType"`
	Status   device.DeviceStatus `json:"status"`
}

// DeviceListResponse 设备列表响应
type DeviceListResponse struct {
	Devices []DeviceInfo `json:"devices"`
	Total   int          `json:"total"`
}

// HandTypeRequest 手型设置请求
type HandTypeRequest struct {
	HandType string `json:"handType" binding:"required,oneof=left right"`
}

// ===== 姿态控制相关模型 =====

// FingerPoseRequest 手指姿态设置请求
type FingerPoseRequest struct {
	Pose []byte `json:"pose" binding:"required,len=6"`
}

// PalmPoseRequest 手掌姿态设置请求
type PalmPoseRequest struct {
	Pose []byte `json:"pose" binding:"required,len=4"`
}

// ===== 动画控制相关模型 =====

// AnimationStartRequest 动画启动请求
type AnimationStartRequest struct {
	Name    string `json:"name" binding:"required"`
	SpeedMs int    `json:"speedMs,omitempty"`
}

// AnimationStatusResponse 动画状态响应
type AnimationStatusResponse struct {
	IsRunning     bool     `json:"isRunning"`
	CurrentName   string   `json:"currentName,omitempty"`
	AvailableList []string `json:"availableList"`
}

// ===== 传感器相关模型 =====

// SensorDataResponse 传感器数据响应
type SensorDataResponse struct {
	SensorID  string         `json:"sensorId"`
	Timestamp time.Time      `json:"timestamp"`
	Values    map[string]any `json:"values"`
}

// SensorListResponse 传感器列表响应
type SensorListResponse struct {
	Sensors []SensorDataResponse `json:"sensors"`
	Total   int                  `json:"total"`
}

// ===== 系统管理相关模型 =====

// SystemStatusResponse 系统状态响应
type SystemStatusResponse struct {
	TotalDevices    int                   `json:"totalDevices"`
	ActiveDevices   int                   `json:"activeDevices"`
	SupportedModels []string              `json:"supportedModels"`
	Devices         map[string]DeviceInfo `json:"devices"`
	Uptime          time.Duration         `json:"uptime"`
}

// SupportedModelsResponse 支持的设备型号响应
type SupportedModelsResponse struct {
	Models []string `json:"models"`
	Total  int      `json:"total"`
}

// HealthResponse 健康检查响应
type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version,omitempty"`
}
