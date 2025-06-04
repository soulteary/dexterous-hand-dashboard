package legacy

import (
	"fmt"
	"net/http"
	"time"

	"hands/config"
	"hands/define"

	"github.com/gin-gonic/gin"
)

// handleHealth 健康检查处理函数
func (s *LegacyServer) handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, define.ApiResponse{
		Status:  "success",
		Message: "CAN Control Service is running",
		Data: map[string]any{
			"timestamp":           time.Now(),
			"availableInterfaces": config.Config.AvailableInterfaces,
			"defaultInterface":    config.Config.DefaultInterface,
			"serviceVersion":      "1.0.0-hand-type-support",
		},
	})
}

// handleInterfaces 获取可用接口列表处理函数
func (s *LegacyServer) handleInterfaces(c *gin.Context) {
	responseData := map[string]any{
		"availableInterfaces": config.Config.AvailableInterfaces,
		"defaultInterface":    config.Config.DefaultInterface,
	}

	c.JSON(http.StatusOK, define.ApiResponse{
		Status: "success",
		Data:   responseData,
	})
}

// handleHandConfigs 获取手型配置处理函数
func (s *LegacyServer) handleHandConfigs(c *gin.Context) {
	allHandConfigs := s.mapper.GetAllHandConfigs()

	result := make(map[string]any)
	for _, ifName := range config.Config.AvailableInterfaces {
		if handConfig, exists := allHandConfigs[ifName]; exists {
			result[ifName] = map[string]any{
				"handType": handConfig.HandType,
				"handId":   handConfig.HandId,
			}
		} else {
			// 返回默认配置
			result[ifName] = map[string]any{
				"handType": "right",
				"handId":   define.HAND_TYPE_RIGHT,
			}
		}
	}

	c.JSON(http.StatusOK, define.ApiResponse{
		Status: "success",
		Data:   result,
	})
}

// handleHandType 手型设置处理函数
func (s *LegacyServer) handleHandType(c *gin.Context) {
	var req HandTypeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, define.ApiResponse{
			Status: "error",
			Error:  "无效的手型设置请求：" + err.Error(),
		})
		return
	}

	// 验证接口
	if !s.mapper.IsValidInterface(req.Interface) {
		c.JSON(http.StatusBadRequest, define.ApiResponse{
			Status: "error",
			Error:  fmt.Sprintf("无效的接口 %s，可用接口: %v", req.Interface, config.Config.AvailableInterfaces),
		})
		return
	}

	// 验证手型 ID
	if req.HandType == "left" && req.HandId != uint32(define.HAND_TYPE_LEFT) {
		req.HandId = uint32(define.HAND_TYPE_LEFT)
	} else if req.HandType == "right" && req.HandId != uint32(define.HAND_TYPE_RIGHT) {
		req.HandId = uint32(define.HAND_TYPE_RIGHT)
	}

	// 设置手型配置
	if err := s.mapper.SetHandConfig(req.Interface, req.HandType, req.HandId); err != nil {
		c.JSON(http.StatusInternalServerError, define.ApiResponse{
			Status: "error",
			Error:  "设置手型失败：" + err.Error(),
		})
		return
	}

	handTypeName := "右手"
	if req.HandType == "left" {
		handTypeName = "左手"
	}

	c.JSON(http.StatusOK, define.ApiResponse{
		Status:  "success",
		Message: fmt.Sprintf("接口 %s 手型已设置为%s (0x%X)", req.Interface, handTypeName, req.HandId),
		Data: map[string]any{
			"interface": req.Interface,
			"handType":  req.HandType,
			"handId":    req.HandId,
		},
	})
}

// handleFingers 手指姿态处理函数
func (s *LegacyServer) handleFingers(c *gin.Context) {
	var req FingerPoseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, define.ApiResponse{
			Status: "error",
			Error:  "无效的手指姿态数据：" + err.Error(),
		})
		return
	}

	// 验证每个值是否在范围内
	for _, v := range req.Pose {
		if v < 0 || v > 255 {
			c.JSON(http.StatusBadRequest, define.ApiResponse{
				Status: "error",
				Error:  "手指姿态值必须在 0-255 范围内",
			})
			return
		}
	}

	// 如果未指定接口，使用默认接口
	if req.Interface == "" {
		req.Interface = config.Config.DefaultInterface
	}

	// 验证接口
	if !s.mapper.IsValidInterface(req.Interface) {
		c.JSON(http.StatusBadRequest, define.ApiResponse{
			Status: "error",
			Error:  fmt.Sprintf("无效的接口 %s，可用接口: %v", req.Interface, config.Config.AvailableInterfaces),
		})
		return
	}

	// 获取对应的设备
	dev, err := s.mapper.GetDeviceForInterface(req.Interface)
	if err != nil {
		c.JSON(http.StatusInternalServerError, define.ApiResponse{
			Status: "error",
			Error:  "获取设备失败：" + err.Error(),
		})
		return
	}

	// 停止当前动画
	if err := s.mapper.StopAllAnimations(req.Interface); err != nil {
		c.JSON(http.StatusInternalServerError, define.ApiResponse{
			Status: "error",
			Error:  "停止动画失败：" + err.Error(),
		})
		return
	}

	// 设置手指姿态
	if err := dev.SetFingerPose(req.Pose); err != nil {
		c.JSON(http.StatusInternalServerError, define.ApiResponse{
			Status: "error",
			Error:  "发送手指姿态失败：" + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, define.ApiResponse{
		Status:  "success",
		Message: "手指姿态指令发送成功",
		Data:    map[string]any{"interface": req.Interface, "pose": req.Pose},
	})
}

// handlePalm 掌部姿态处理函数
func (s *LegacyServer) handlePalm(c *gin.Context) {
	var req PalmPoseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, define.ApiResponse{
			Status: "error",
			Error:  "无效的掌部姿态数据：" + err.Error(),
		})
		return
	}

	// 验证每个值是否在范围内
	for _, v := range req.Pose {
		if v < 0 || v > 255 {
			c.JSON(http.StatusBadRequest, define.ApiResponse{
				Status: "error",
				Error:  "掌部姿态值必须在 0-255 范围内",
			})
			return
		}
	}

	// 如果未指定接口，使用默认接口
	if req.Interface == "" {
		req.Interface = config.Config.DefaultInterface
	}

	// 验证接口
	if !s.mapper.IsValidInterface(req.Interface) {
		c.JSON(http.StatusBadRequest, define.ApiResponse{
			Status: "error",
			Error:  fmt.Sprintf("无效的接口 %s，可用接口: %v", req.Interface, config.Config.AvailableInterfaces),
		})
		return
	}

	// 获取对应的设备
	dev, err := s.mapper.GetDeviceForInterface(req.Interface)
	if err != nil {
		c.JSON(http.StatusInternalServerError, define.ApiResponse{
			Status: "error",
			Error:  "获取设备失败：" + err.Error(),
		})
		return
	}

	// 停止当前动画
	if err := s.mapper.StopAllAnimations(req.Interface); err != nil {
		c.JSON(http.StatusInternalServerError, define.ApiResponse{
			Status: "error",
			Error:  "停止动画失败：" + err.Error(),
		})
		return
	}

	// 设置掌部姿态
	if err := dev.SetPalmPose(req.Pose); err != nil {
		c.JSON(http.StatusInternalServerError, define.ApiResponse{
			Status: "error",
			Error:  "发送掌部姿态失败：" + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, define.ApiResponse{
		Status:  "success",
		Message: "掌部姿态指令发送成功",
		Data:    map[string]any{"interface": req.Interface, "pose": req.Pose},
	})
}

// handlePreset 预设姿势处理函数
func (s *LegacyServer) handlePreset(c *gin.Context) {
	pose := c.Param("pose")

	// 从查询参数获取接口名称和手型
	ifName := c.Query("interface")
	// handType := c.Query("handType") // TODO: 旧版 API 中声明但未使用，先放着，等 reivew 时候看看

	if ifName == "" {
		ifName = config.Config.DefaultInterface
	}

	// 验证接口
	if !s.mapper.IsValidInterface(ifName) {
		c.JSON(http.StatusBadRequest, define.ApiResponse{
			Status: "error",
			Error:  fmt.Sprintf("无效的接口 %s，可用接口: %v", ifName, config.Config.AvailableInterfaces),
		})
		return
	}

	// 获取对应的设备
	dev, err := s.mapper.GetDeviceForInterface(ifName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, define.ApiResponse{
			Status: "error",
			Error:  "获取设备失败：" + err.Error(),
		})
		return
	}

	// 停止当前动画
	if err := s.mapper.StopAllAnimations(ifName); err != nil {
		c.JSON(http.StatusInternalServerError, define.ApiResponse{
			Status: "error",
			Error:  "停止动画失败：" + err.Error(),
		})
		return
	}

	// 获取预设姿势详细信息（用于返回具体参数）
	presetDetails, exists := dev.GetPresetDetails(pose)
	if !exists {
		c.JSON(http.StatusBadRequest, define.ApiResponse{
			Status: "error",
			Error:  "无效的预设姿势",
		})
		return
	}

	// 使用设备的预设姿势方法
	if err := dev.ExecutePreset(pose); err != nil {
		c.JSON(http.StatusBadRequest, define.ApiResponse{
			Status: "error",
			Error:  "无效的预设姿势",
		})
		return
	}

	// 获取预设姿势的描述
	description := dev.GetPresetDescription(pose)
	message := fmt.Sprintf("已设置预设姿势: %s", pose)
	if description != "" {
		message = fmt.Sprintf("已设置%s", description)
	}

	c.JSON(http.StatusOK, define.ApiResponse{
		Status:  "success",
		Message: message,
		Data:    map[string]any{"interface": ifName, "pose": presetDetails.FingerPose},
	})
}

// handleAnimation 动画控制处理函数
func (s *LegacyServer) handleAnimation(c *gin.Context) {
	var req AnimationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, define.ApiResponse{
			Status: "error",
			Error:  "无效的动画请求：" + err.Error(),
		})
		return
	}

	// 如果未指定接口，使用默认接口
	if req.Interface == "" {
		req.Interface = config.Config.DefaultInterface
	}

	// 验证接口
	if !s.mapper.IsValidInterface(req.Interface) {
		c.JSON(http.StatusBadRequest, define.ApiResponse{
			Status: "error",
			Error:  fmt.Sprintf("无效的接口 %s，可用接口: %v", req.Interface, config.Config.AvailableInterfaces),
		})
		return
	}

	// 获取对应的设备
	dev, err := s.mapper.GetDeviceForInterface(req.Interface)
	if err != nil {
		c.JSON(http.StatusInternalServerError, define.ApiResponse{
			Status: "error",
			Error:  "获取设备失败：" + err.Error(),
		})
		return
	}

	// 获取动画引擎
	animEngine := dev.GetAnimationEngine()

	// 停止当前动画
	if err := s.mapper.StopAllAnimations(req.Interface); err != nil {
		c.JSON(http.StatusInternalServerError, define.ApiResponse{
			Status: "error",
			Error:  "停止动画失败：" + err.Error(),
		})
		return
	}

	// 如果是停止命令，直接返回
	if req.Type == "stop" {
		c.JSON(http.StatusOK, define.ApiResponse{
			Status:  "success",
			Message: fmt.Sprintf("%s 动画已停止", req.Interface),
		})
		return
	}

	// 处理速度参数
	if req.Speed <= 0 {
		req.Speed = 500 // 默认速度
	}

	// 根据类型启动动画
	switch req.Type {
	case "wave":
		if err := animEngine.Start("wave", req.Speed); err != nil {
			c.JSON(http.StatusInternalServerError, define.ApiResponse{
				Status: "error",
				Error:  fmt.Sprintf("启动波浪动画失败：%v", err),
			})
			return
		}
		c.JSON(http.StatusOK, define.ApiResponse{
			Status:  "success",
			Message: fmt.Sprintf("%s 波浪动画已启动", req.Interface),
			Data:    map[string]any{"interface": req.Interface, "speed": req.Speed},
		})
	case "sway":
		if err := animEngine.Start("sway", req.Speed); err != nil {
			c.JSON(http.StatusInternalServerError, define.ApiResponse{
				Status: "error",
				Error:  fmt.Sprintf("启动横向摆动动画失败：%v", err),
			})
			return
		}
		c.JSON(http.StatusOK, define.ApiResponse{
			Status:  "success",
			Message: fmt.Sprintf("%s 横向摆动动画已启动", req.Interface),
			Data:    map[string]any{"interface": req.Interface, "speed": req.Speed},
		})
	default:
		c.JSON(http.StatusBadRequest, define.ApiResponse{
			Status: "error",
			Error:  "无效的动画类型",
		})
	}
}

// handleSensors 获取传感器数据处理函数
func (s *LegacyServer) handleSensors(c *gin.Context) {
	// 从查询参数获取接口名称
	ifName := c.Query("interface")

	if ifName != "" {
		// 验证接口
		if !s.mapper.IsValidInterface(ifName) {
			c.JSON(http.StatusBadRequest, define.ApiResponse{
				Status: "error",
				Error:  fmt.Sprintf("无效的接口 %s，可用接口: %v", ifName, config.Config.AvailableInterfaces),
			})
			return
		}

		// 获取对应的设备
		dev, err := s.mapper.GetDeviceForInterface(ifName)
		if err != nil {
			c.JSON(http.StatusInternalServerError, define.ApiResponse{
				Status: "error",
				Error:  "获取设备失败：" + err.Error(),
			})
			return
		}

		sensorData, err := dev.ReadSensorData()
		if err != nil {
			c.JSON(http.StatusInternalServerError, define.ApiResponse{
				Status: "error",
				Error:  "获取传感器数据失败：" + err.Error(),
			})
		}

		c.JSON(http.StatusOK, define.ApiResponse{
			Status: "success",
			Data:   sensorData.Values(),
		})
	} else {
		// 返回所有接口的传感器数据
		allSensorData := make(map[string]any)

		for _, ifName := range config.Config.AvailableInterfaces {
			// 获取对应的设备
			dev, err := s.mapper.GetDeviceForInterface(ifName)
			if err != nil {
				allSensorData[ifName] = map[string]any{"error": "设备不可用：" + err.Error()}
				continue
			}

			sensorData, err := dev.ReadSensorData()
			if err != nil {
				allSensorData[ifName] = map[string]any{"error": "设备不可用：" + err.Error()}
				continue
			}

			allSensorData[ifName] = sensorData.Values()
		}

		c.JSON(http.StatusOK, define.ApiResponse{
			Status: "success",
			Data:   allSensorData,
		})
	}
}

// handleStatus 系统状态处理函数
func (s *LegacyServer) handleStatus(c *gin.Context) {
	// 构建动画状态
	animationStatus := make(map[string]bool)

	// 获取手型配置
	allHandConfigs := s.mapper.GetAllHandConfigs()
	handConfigsData := make(map[string]any)

	// 构建接口状态信息
	interfaceStatuses := make(map[string]any)

	// 检查 CAN 服务状态 - 通过尝试获取设备状态来判断
	canStatus := make(map[string]bool)

	for ifName, handConfig := range allHandConfigs {
		handConfigsData[ifName] = map[string]any{
			"handType": handConfig.HandType,
			"handId":   handConfig.HandId,
		}
	}

	for _, ifName := range config.Config.AvailableInterfaces {
		// 获取对应的设备
		dev, err := s.mapper.GetDeviceForInterface(ifName)
		if err != nil {
			// 设备不可用
			animationStatus[ifName] = false
			canStatus[ifName] = false
			handConfigsData[ifName] = map[string]any{
				"handType": "right",
				"handId":   define.HAND_TYPE_RIGHT,
			}
		} else {
			// 获取动画状态
			animEngine := dev.GetAnimationEngine()
			animationStatus[ifName] = animEngine.IsRunning()

			// 获取设备状态来判断 CAN 服务状态
			rawCanStatus, err := dev.GetCanStatus()
			if err != nil {
				canStatus[ifName] = false
			} else {
				canStatus[ifName] = rawCanStatus[ifName]
			}
		}

		// 构建接口状态
		interfaceStatuses[ifName] = map[string]any{
			"active":          canStatus[ifName],
			"animationActive": animationStatus[ifName],
			"handConfig":      handConfigsData[ifName],
		}
	}

	// 计算活跃接口数量
	activeInterfacesCount := 0
	for _, active := range canStatus {
		if active {
			activeInterfacesCount++
		}
	}

	c.JSON(http.StatusOK, define.ApiResponse{
		Status: "success",
		Data: map[string]any{
			"interfaces":          interfaceStatuses,
			"uptime":              time.Since(s.startTime).String(),
			"canServiceURL":       config.Config.CanServiceURL,
			"defaultInterface":    config.Config.DefaultInterface,
			"availableInterfaces": config.Config.AvailableInterfaces,
			"activeInterfaces":    activeInterfacesCount,
			"handConfigs":         handConfigsData,
		},
	})
}
