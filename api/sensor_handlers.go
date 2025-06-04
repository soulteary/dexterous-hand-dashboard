package api

import (
	"fmt"
	"net/http"

	"hands/device"

	"github.com/gin-gonic/gin"
)

// handleGetSensors 获取所有传感器数据
func (s *Server) handleGetSensors(c *gin.Context) {
	deviceId := c.Param("deviceId")

	// 获取设备
	dev, err := s.deviceManager.GetDevice(deviceId)
	if err != nil {
		c.JSON(http.StatusNotFound, ApiResponse{
			Status: "error",
			Error:  fmt.Sprintf("设备 %s 不存在", deviceId),
		})
		return
	}

	sensorData, err := dev.ReadSensorData()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ApiResponse{
			Status: "error",
			Error:  fmt.Sprintf("读取传感器数据失败：%v", err),
		})
	}

	c.JSON(http.StatusOK, ApiResponse{
		Status: "success",
		Data:   sensorData.Values(),
	})
}

// handleGetDeviceStatus 获取设备状态
func (s *Server) handleGetDeviceStatus(c *gin.Context) {
	deviceId := c.Param("deviceId")

	// 获取设备
	dev, err := s.deviceManager.GetDevice(deviceId)
	if err != nil {
		c.JSON(http.StatusNotFound, ApiResponse{
			Status: "error",
			Error:  fmt.Sprintf("设备 %s 不存在", deviceId),
		})
		return
	}

	// 获取设备状态
	status, err := dev.GetStatus()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ApiResponse{
			Status: "error",
			Error:  fmt.Sprintf("获取设备状态失败：%v", err),
		})
		return
	}

	// 获取动画引擎状态
	animEngine := dev.GetAnimationEngine()
	animationStatus := map[string]any{
		"isRunning": animEngine.IsRunning(),
	}

	// 获取传感器组件数量
	sensorComponents := dev.GetComponents(device.SensorComponent)

	// 构建详细的设备状态响应
	deviceInfo := DeviceInfo{
		ID:       dev.GetID(),
		Model:    dev.GetModel(),
		HandType: dev.GetHandType().String(),
		Status:   status,
	}

	// 扩展状态信息
	extendedStatus := map[string]any{
		"device":      deviceInfo,
		"animation":   animationStatus,
		"sensorCount": len(sensorComponents),
		"lastUpdate":  status.LastUpdate,
	}

	c.JSON(http.StatusOK, ApiResponse{
		Status: "success",
		Data:   extendedStatus,
	})
}
