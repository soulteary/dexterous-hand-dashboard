package api

import (
	"fmt"
	"hands/config"
	"hands/define"
	"hands/hands"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// 手型设置处理函数
func HandleHandType(c *gin.Context) {
	var req HandTypeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, define.ApiResponse{
			Status: "error",
			Error:  "无效的手型设置请求：" + err.Error(),
		})
		return
	}

	// 验证接口
	if !config.IsValidInterface(req.Interface) {
		c.JSON(http.StatusBadRequest, define.ApiResponse{
			Status: "error",
			Error:  fmt.Sprintf("无效的接口 %s，可用接口: %v", req.Interface, config.Config.AvailableInterfaces),
		})
		return
	}

	// 验证手型 ID
	if req.HandType == "left" && req.HandId != define.HAND_TYPE_LEFT {
		req.HandId = define.HAND_TYPE_LEFT
	} else if req.HandType == "right" && req.HandId != define.HAND_TYPE_RIGHT {
		req.HandId = define.HAND_TYPE_RIGHT
	}

	// 设置手型配置
	hands.SetHandConfig(req.Interface, req.HandType, req.HandId)

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

// 手指姿态处理函数
func HandleFingers(c *gin.Context) {
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
	if !config.IsValidInterface(req.Interface) {
		c.JSON(http.StatusBadRequest, define.ApiResponse{
			Status: "error",
			Error:  fmt.Sprintf("无效的接口 %s，可用接口: %v", req.Interface, config.Config.AvailableInterfaces),
		})
		return
	}

	hands.StopAllAnimations(req.Interface)

	if err := hands.SendFingerPose(req.Interface, req.Pose, req.HandType, req.HandId); err != nil {
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

// 掌部姿态处理函数
func HandlePalm(c *gin.Context) {
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
	if !config.IsValidInterface(req.Interface) {
		c.JSON(http.StatusBadRequest, define.ApiResponse{
			Status: "error",
			Error:  fmt.Sprintf("无效的接口 %s，可用接口: %v", req.Interface, config.Config.AvailableInterfaces),
		})
		return
	}

	hands.StopAllAnimations(req.Interface)

	if err := hands.SendPalmPose(req.Interface, req.Pose, req.HandType, req.HandId); err != nil {
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

// 预设姿势处理函数
func HandlePreset(c *gin.Context) {
	pose := c.Param("pose")

	// 从查询参数获取接口名称和手型
	ifName := c.Query("interface")
	handType := c.Query("handType")

	if ifName == "" {
		ifName = config.Config.DefaultInterface
	}

	// 验证接口
	if !config.IsValidInterface(ifName) {
		c.JSON(http.StatusBadRequest, define.ApiResponse{
			Status: "error",
			Error:  fmt.Sprintf("无效的接口 %s，可用接口: %v", ifName, config.Config.AvailableInterfaces),
		})
		return
	}

	hands.StopAllAnimations(ifName)

	var fingerPose []byte
	var message string

	switch pose {
	case "fist":
		fingerPose = []byte{64, 64, 64, 64, 64, 64}
		message = "已设置握拳姿势"
	case "open":
		fingerPose = []byte{192, 192, 192, 192, 192, 192}
		message = "已设置完全张开姿势"
	case "pinch":
		fingerPose = []byte{120, 120, 64, 64, 64, 64}
		message = "已设置捏取姿势"
	case "thumbsup":
		fingerPose = []byte{64, 192, 192, 192, 192, 64}
		message = "已设置竖起大拇指姿势"
	case "point":
		fingerPose = []byte{192, 64, 192, 192, 192, 64}
		message = "已设置食指指点姿势"
	// 数字手势
	case "1":
		fingerPose = []byte{192, 64, 192, 192, 192, 64}
		message = "已设置数字 1 手势"
	case "2":
		fingerPose = []byte{192, 64, 64, 192, 192, 64}
		message = "已设置数字 2 手势"
	case "3":
		fingerPose = []byte{192, 64, 64, 64, 192, 64}
		message = "已设置数字 3 手势"
	case "4":
		fingerPose = []byte{192, 64, 64, 64, 64, 64}
		message = "已设置数字 4 手势"
	case "5":
		fingerPose = []byte{192, 192, 192, 192, 192, 192}
		message = "已设置数字 5 手势"
	case "6":
		fingerPose = []byte{64, 192, 192, 192, 192, 64}
		message = "已设置数字 6 手势"
	case "7":
		fingerPose = []byte{64, 64, 192, 192, 192, 64}
		message = "已设置数字 7 手势"
	case "8":
		fingerPose = []byte{64, 64, 64, 192, 192, 64}
		message = "已设置数字 8 手势"
	case "9":
		fingerPose = []byte{64, 64, 64, 64, 192, 64}
		message = "已设置数字 9 手势"
	default:
		c.JSON(http.StatusBadRequest, define.ApiResponse{
			Status: "error",
			Error:  "无效的预设姿势",
		})
		return
	}

	// 解析手型 ID（从查询参数或使用接口配置）
	handId := uint32(0)
	if handType != "" {
		handId = hands.ParseHandType(handType, 0, ifName)
	}

	if err := hands.SendFingerPose(ifName, fingerPose, handType, handId); err != nil {
		c.JSON(http.StatusInternalServerError, define.ApiResponse{
			Status: "error",
			Error:  "设置预设姿势失败：" + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, define.ApiResponse{
		Status:  "success",
		Message: message,
		Data:    map[string]any{"interface": ifName, "pose": fingerPose},
	})
}

// 动画控制处理函数
func HandleAnimation(c *gin.Context) {
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
	if !config.IsValidInterface(req.Interface) {
		c.JSON(http.StatusBadRequest, define.ApiResponse{
			Status: "error",
			Error:  fmt.Sprintf("无效的接口 %s，可用接口: %v", req.Interface, config.Config.AvailableInterfaces),
		})
		return
	}

	// 停止当前动画
	hands.StopAllAnimations(req.Interface)

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
		hands.StartWaveAnimation(req.Interface, req.Speed, req.HandType, req.HandId)
		c.JSON(http.StatusOK, define.ApiResponse{
			Status:  "success",
			Message: fmt.Sprintf("%s 波浪动画已启动", req.Interface),
			Data:    map[string]any{"interface": req.Interface, "speed": req.Speed},
		})
	case "sway":
		hands.StartSwayAnimation(req.Interface, req.Speed, req.HandType, req.HandId)
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

// 获取传感器数据处理函数
func HandleSensors(c *gin.Context) {
	// 从查询参数获取接口名称
	ifName := c.Query("interface")

	hands.SensorMutex.RLock()
	defer hands.SensorMutex.RUnlock()

	if ifName != "" {
		// 验证接口
		if !config.IsValidInterface(ifName) {
			c.JSON(http.StatusBadRequest, define.ApiResponse{
				Status: "error",
				Error:  fmt.Sprintf("无效的接口 %s，可用接口: %v", ifName, config.Config.AvailableInterfaces),
			})
			return
		}

		// 请求特定接口的数据
		if sensorData, ok := hands.SensorDataMap[ifName]; ok {
			c.JSON(http.StatusOK, define.ApiResponse{
				Status: "success",
				Data:   sensorData,
			})
		} else {
			c.JSON(http.StatusInternalServerError, define.ApiResponse{
				Status: "error",
				Error:  "传感器数据不存在",
			})
		}
	} else {
		// 返回所有接口的数据
		c.JSON(http.StatusOK, define.ApiResponse{
			Status: "success",
			Data:   hands.SensorDataMap,
		})
	}
}

// 系统状态处理函数
func HandleStatus(c *gin.Context) {
	hands.AnimationMutex.Lock()
	animationStatus := make(map[string]bool)
	for _, ifName := range config.Config.AvailableInterfaces {
		animationStatus[ifName] = hands.AnimationActive[ifName]
	}
	hands.AnimationMutex.Unlock()

	// 检查 CAN 服务状态
	canStatus := hands.CheckCanServiceStatus()

	// 获取手型配置
	hands.HandConfigMutex.RLock()
	handConfigsData := make(map[string]any)
	for ifName, handConfig := range hands.HandConfigs {
		handConfigsData[ifName] = map[string]any{
			"handType": handConfig.HandType,
			"handId":   handConfig.HandId,
		}
	}
	hands.HandConfigMutex.RUnlock()

	interfaceStatuses := make(map[string]any)
	for _, ifName := range config.Config.AvailableInterfaces {
		interfaceStatuses[ifName] = map[string]any{
			"active":          canStatus[ifName],
			"animationActive": animationStatus[ifName],
			"handConfig":      handConfigsData[ifName],
		}
	}

	c.JSON(http.StatusOK, define.ApiResponse{
		Status: "success",
		Data: map[string]any{
			"interfaces":          interfaceStatuses,
			"uptime":              time.Since(ServerStartTime).String(),
			"canServiceURL":       config.Config.CanServiceURL,
			"defaultInterface":    config.Config.DefaultInterface,
			"availableInterfaces": config.Config.AvailableInterfaces,
			"activeInterfaces":    len(canStatus),
			"handConfigs":         handConfigsData,
		},
	})
}

// 获取可用接口列表处理函数
func HandleInterfaces(c *gin.Context) {
	responseData := map[string]any{
		"availableInterfaces": config.Config.AvailableInterfaces,
		"defaultInterface":    config.Config.DefaultInterface,
	}

	c.JSON(http.StatusOK, define.ApiResponse{
		Status: "success",
		Data:   responseData,
	})
}

// 获取手型配置处理函数
func HandleHandConfigs(c *gin.Context) {
	hands.HandConfigMutex.RLock()
	defer hands.HandConfigMutex.RUnlock()

	result := make(map[string]any)
	for _, ifName := range config.Config.AvailableInterfaces {
		if handConfig, exists := hands.HandConfigs[ifName]; exists {
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

// 健康检查处理函数
func HandleHealth(c *gin.Context) {
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
