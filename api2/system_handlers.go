package api2

import (
	"net/http"
	"time"

	"hands/device"

	"github.com/gin-gonic/gin"
)

// handleGetSupportedModels 获取支持的设备型号
func (s *Server) handleGetSupportedModels(c *gin.Context) {
	// 获取支持的设备型号列表
	models := device.GetSupportedModels()

	response := SupportedModelsResponse{
		Models: models,
		Total:  len(models),
	}

	c.JSON(http.StatusOK, ApiResponse{
		Status: "success",
		Data:   response,
	})
}

// handleGetSystemStatus 获取系统状态
func (s *Server) handleGetSystemStatus(c *gin.Context) {
	// 获取所有设备
	devices := s.deviceManager.GetAllDevices()

	// 统计设备信息
	totalDevices := len(devices)
	activeDevices := 0
	deviceInfos := make(map[string]DeviceInfo)

	for _, dev := range devices {
		status, err := dev.GetStatus()
		if err != nil {
			// 如果获取状态失败，使用默认状态
			status = device.DeviceStatus{
				IsConnected: false,
				IsActive:    false,
				ErrorCount:  1,
				LastError:   err.Error(),
			}
		}

		if status.IsActive {
			activeDevices++
		}

		deviceInfo := DeviceInfo{
			ID:       dev.GetID(),
			Model:    dev.GetModel(),
			HandType: dev.GetHandType().String(),
			Status:   status,
		}
		deviceInfos[dev.GetID()] = deviceInfo
	}

	// 获取支持的设备型号
	supportedModels := device.GetSupportedModels()

	// 计算系统运行时间
	uptime := time.Since(s.startTime)

	response := SystemStatusResponse{
		TotalDevices:    totalDevices,
		ActiveDevices:   activeDevices,
		SupportedModels: supportedModels,
		Devices:         deviceInfos,
		Uptime:          uptime,
	}

	c.JSON(http.StatusOK, ApiResponse{
		Status: "success",
		Data:   response,
	})
}

// handleHealthCheck 健康检查
func (s *Server) handleHealthCheck(c *gin.Context) {
	// 执行基本的健康检查
	status := "healthy"

	// 检查设备管理器是否正常
	if s.deviceManager == nil {
		status = "unhealthy"
	}

	// 可以添加更多健康检查逻辑，比如：
	// - 检查关键服务是否可用
	// - 检查数据库连接
	// - 检查外部依赖

	response := HealthResponse{
		Status:    status,
		Timestamp: time.Now(),
		Version:   s.version,
	}

	// 根据健康状态返回相应的 HTTP 状态码
	httpStatus := http.StatusOK
	if status != "healthy" {
		httpStatus = http.StatusServiceUnavailable
	}

	c.JSON(httpStatus, ApiResponse{
		Status: "success",
		Data:   response,
	})
}
