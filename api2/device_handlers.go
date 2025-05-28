package api2

import (
	"fmt"
	"net/http"

	"hands/define"
	"hands/device"

	"github.com/gin-gonic/gin"
)

// handleGetDevices 获取所有设备列表
func (s *Server) handleGetDevices(c *gin.Context) {
	devices := s.deviceManager.GetAllDevices()

	deviceInfos := make([]DeviceInfo, 0, len(devices))
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

		deviceInfo := DeviceInfo{
			ID:       dev.GetID(),
			Model:    dev.GetModel(),
			HandType: dev.GetHandType().String(),
			Status:   status,
		}
		deviceInfos = append(deviceInfos, deviceInfo)
	}

	response := DeviceListResponse{
		Devices: deviceInfos,
		Total:   len(deviceInfos),
	}

	c.JSON(http.StatusOK, ApiResponse{
		Status: "success",
		Data:   response,
	})
}

// handleCreateDevice 创建新设备
func (s *Server) handleCreateDevice(c *gin.Context) {
	var req DeviceCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ApiResponse{
			Status: "error",
			Error:  "无效的设备创建请求：" + err.Error(),
		})
		return
	}

	// 检查设备是否已存在
	if _, err := s.deviceManager.GetDevice(req.ID); err == nil {
		c.JSON(http.StatusConflict, ApiResponse{
			Status: "error",
			Error:  fmt.Sprintf("设备 %s 已存在", req.ID),
		})
		return
	}

	// 准备设备配置
	config := req.Config
	if config == nil {
		config = make(map[string]any)
	}
	config["id"] = req.ID

	// 设置手型
	if req.HandType != "" {
		config["hand_type"] = req.HandType
	}

	// 创建设备实例
	dev, err := device.CreateDevice(req.Model, config)
	if err != nil {
		c.JSON(http.StatusBadRequest, ApiResponse{
			Status: "error",
			Error:  fmt.Sprintf("创建设备失败：%v", err),
		})
		return
	}

	// 注册设备到管理器
	if err := s.deviceManager.RegisterDevice(dev); err != nil {
		c.JSON(http.StatusInternalServerError, ApiResponse{
			Status: "error",
			Error:  fmt.Sprintf("注册设备失败：%v", err),
		})
		return
	}

	// 获取设备状态
	status, err := dev.GetStatus()
	if err != nil {
		status = device.DeviceStatus{
			IsConnected: false,
			IsActive:    false,
			ErrorCount:  1,
			LastError:   err.Error(),
		}
	}

	deviceInfo := DeviceInfo{
		ID:       dev.GetID(),
		Model:    dev.GetModel(),
		HandType: dev.GetHandType().String(),
		Status:   status,
	}

	c.JSON(http.StatusCreated, ApiResponse{
		Status:  "success",
		Message: fmt.Sprintf("设备 %s 创建成功", req.ID),
		Data:    deviceInfo,
	})
}

// handleGetDevice 获取设备详情
func (s *Server) handleGetDevice(c *gin.Context) {
	deviceId := c.Param("deviceId")

	dev, err := s.deviceManager.GetDevice(deviceId)
	if err != nil {
		c.JSON(http.StatusNotFound, ApiResponse{
			Status: "error",
			Error:  fmt.Sprintf("设备 %s 不存在", deviceId),
		})
		return
	}

	status, err := dev.GetStatus()
	if err != nil {
		status = device.DeviceStatus{
			IsConnected: false,
			IsActive:    false,
			ErrorCount:  1,
			LastError:   err.Error(),
		}
	}

	deviceInfo := DeviceInfo{
		ID:       dev.GetID(),
		Model:    dev.GetModel(),
		HandType: dev.GetHandType().String(),
		Status:   status,
	}

	c.JSON(http.StatusOK, ApiResponse{
		Status: "success",
		Data:   deviceInfo,
	})
}

// handleDeleteDevice 删除设备
func (s *Server) handleDeleteDevice(c *gin.Context) {
	deviceId := c.Param("deviceId")

	// 检查设备是否存在
	dev, err := s.deviceManager.GetDevice(deviceId)
	if err != nil {
		c.JSON(http.StatusNotFound, ApiResponse{
			Status: "error",
			Error:  fmt.Sprintf("设备 %s 不存在", deviceId),
		})
		return
	}

	// 停止设备的动画（如果正在运行）
	animEngine := dev.GetAnimationEngine()
	if animEngine.IsRunning() {
		if err := animEngine.Stop(); err != nil {
			// 记录错误但不阻止删除
			fmt.Printf("警告：停止设备 %s 动画时出错：%v\n", deviceId, err)
		}
	}

	// 从管理器中移除设备
	if err := s.deviceManager.RemoveDevice(deviceId); err != nil {
		c.JSON(http.StatusInternalServerError, ApiResponse{
			Status: "error",
			Error:  fmt.Sprintf("删除设备失败：%v", err),
		})
		return
	}

	c.JSON(http.StatusOK, ApiResponse{
		Status:  "success",
		Message: fmt.Sprintf("设备 %s 已删除", deviceId),
	})
}

// handleSetHandType 设置设备手型
func (s *Server) handleSetHandType(c *gin.Context) {
	deviceId := c.Param("deviceId")

	var req HandTypeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ApiResponse{
			Status: "error",
			Error:  "无效的手型设置请求：" + err.Error(),
		})
		return
	}

	// 获取设备
	dev, err := s.deviceManager.GetDevice(deviceId)
	if err != nil {
		c.JSON(http.StatusNotFound, ApiResponse{
			Status: "error",
			Error:  fmt.Sprintf("设备 %s 不存在", deviceId),
		})
		return
	}

	// 转换手型字符串为枚举
	var handType define.HandType
	handType = define.HandTypeFromString(req.HandType)
	if handType == define.HAND_TYPE_UNKNOWN {
		c.JSON(http.StatusBadRequest, ApiResponse{
			Status: "error",
			Error:  "无效的手型，必须是 'left' 或 'right'",
		})
		return
	}

	// 设置手型
	if err := dev.SetHandType(handType); err != nil {
		c.JSON(http.StatusInternalServerError, ApiResponse{
			Status: "error",
			Error:  fmt.Sprintf("设置手型失败：%v", err),
		})
		return
	}

	c.JSON(http.StatusOK, ApiResponse{
		Status:  "success",
		Message: fmt.Sprintf("设备 %s 手型已设置为 %s", deviceId, req.HandType),
		Data: map[string]any{
			"deviceId": deviceId,
			"handType": req.HandType,
		},
	})
}
