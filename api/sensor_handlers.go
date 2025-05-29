package api

import (
	"fmt"
	"net/http"
	"time"

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

	// 获取设备的传感器组件
	sensorComponents := dev.GetComponents(device.SensorComponent)

	sensors := make([]SensorDataResponse, 0, len(sensorComponents))

	// 遍历所有传感器组件，读取数据
	for _, component := range sensorComponents {
		sensorId := component.GetID()

		// 读取传感器数据
		sensorData, err := dev.ReadSensorData(sensorId)
		if err != nil {
			// 如果读取失败，创建一个错误状态的传感器数据
			sensors = append(sensors, SensorDataResponse{
				SensorID:  sensorId,
				Timestamp: time.Now(),
				Values: map[string]any{
					"error":  err.Error(),
					"status": "error",
				},
			})
			continue
		}

		// 转换为响应格式
		sensorResponse := SensorDataResponse{
			SensorID:  sensorData.SensorID(),
			Timestamp: sensorData.Timestamp(),
			Values:    sensorData.Values(),
		}
		sensors = append(sensors, sensorResponse)
	}

	response := SensorListResponse{
		Sensors: sensors,
		Total:   len(sensors),
	}

	c.JSON(http.StatusOK, ApiResponse{
		Status: "success",
		Data:   response,
	})
}

// handleGetSensorData 获取特定传感器数据
func (s *Server) handleGetSensorData(c *gin.Context) {
	deviceId := c.Param("deviceId")
	sensorId := c.Param("sensorId")

	// 获取设备
	dev, err := s.deviceManager.GetDevice(deviceId)
	if err != nil {
		c.JSON(http.StatusNotFound, ApiResponse{
			Status: "error",
			Error:  fmt.Sprintf("设备 %s 不存在", deviceId),
		})
		return
	}

	// 验证传感器是否存在
	sensorComponents := dev.GetComponents(device.SensorComponent)
	sensorExists := false
	for _, component := range sensorComponents {
		if component.GetID() == sensorId {
			sensorExists = true
			break
		}
	}

	if !sensorExists {
		c.JSON(http.StatusNotFound, ApiResponse{
			Status: "error",
			Error:  fmt.Sprintf("设备 %s 上不存在传感器 %s", deviceId, sensorId),
		})
		return
	}

	// 读取传感器数据
	sensorData, err := dev.ReadSensorData(sensorId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ApiResponse{
			Status: "error",
			Error:  fmt.Sprintf("读取传感器 %s 数据失败：%v", sensorId, err),
		})
		return
	}

	// 转换为响应格式
	response := SensorDataResponse{
		SensorID:  sensorData.SensorID(),
		Timestamp: sensorData.Timestamp(),
		Values:    sensorData.Values(),
	}

	c.JSON(http.StatusOK, ApiResponse{
		Status: "success",
		Data:   response,
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
