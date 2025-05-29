package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// handleGetAnimations 获取可用动画列表
func (s *Server) handleGetAnimations(c *gin.Context) {
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

	// 获取动画引擎
	animEngine := dev.GetAnimationEngine()

	// 获取已注册的动画列表
	availableAnimations := animEngine.GetRegisteredAnimations()

	// 获取当前动画状态
	isRunning := animEngine.IsRunning()
	currentName := animEngine.GetCurrentAnimation()

	response := AnimationStatusResponse{
		IsRunning:     isRunning,
		CurrentName:   currentName,
		AvailableList: availableAnimations,
	}

	c.JSON(http.StatusOK, ApiResponse{
		Status: "success",
		Data:   response,
	})
}

// handleStartAnimation 启动动画
func (s *Server) handleStartAnimation(c *gin.Context) {
	deviceId := c.Param("deviceId")

	var req AnimationStartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ApiResponse{
			Status: "error",
			Error:  "无效的动画请求：" + err.Error(),
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

	// 获取动画引擎
	animEngine := dev.GetAnimationEngine()

	// 验证动画名称是否已注册
	availableAnimations := animEngine.GetRegisteredAnimations()
	validAnimation := false
	for _, name := range availableAnimations {
		if name == req.Name {
			validAnimation = true
			break
		}
	}

	if !validAnimation {
		c.JSON(http.StatusBadRequest, ApiResponse{
			Status: "error",
			Error:  fmt.Sprintf("无效的动画类型：%s，可用动画：%v", req.Name, availableAnimations),
		})
		return
	}

	// 处理速度参数
	speedMs := req.SpeedMs
	if speedMs <= 0 {
		speedMs = 500 // 默认速度
	}

	// 启动动画
	if err := animEngine.Start(req.Name, speedMs); err != nil {
		c.JSON(http.StatusInternalServerError, ApiResponse{
			Status: "error",
			Error:  fmt.Sprintf("启动动画失败：%v", err),
		})
		return
	}

	c.JSON(http.StatusOK, ApiResponse{
		Status:  "success",
		Message: fmt.Sprintf("设备 %s 的 %s 动画已启动", deviceId, req.Name),
		Data: map[string]any{
			"deviceId": deviceId,
			"name":     req.Name,
			"speedMs":  speedMs,
		},
	})
}

// handleStopAnimation 停止动画
func (s *Server) handleStopAnimation(c *gin.Context) {
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

	// 获取动画引擎
	animEngine := dev.GetAnimationEngine()

	// 检查是否有动画在运行
	if !animEngine.IsRunning() {
		c.JSON(http.StatusOK, ApiResponse{
			Status:  "success",
			Message: fmt.Sprintf("设备 %s 当前没有动画在运行", deviceId),
			Data: map[string]any{
				"deviceId": deviceId,
			},
		})
		return
	}

	// 停止动画
	if err := animEngine.Stop(); err != nil {
		c.JSON(http.StatusInternalServerError, ApiResponse{
			Status: "error",
			Error:  fmt.Sprintf("停止动画失败：%v", err),
		})
		return
	}

	c.JSON(http.StatusOK, ApiResponse{
		Status:  "success",
		Message: fmt.Sprintf("设备 %s 的动画已停止", deviceId),
		Data: map[string]any{
			"deviceId": deviceId,
		},
	})
}

// handleAnimationStatus 获取动画状态
func (s *Server) handleAnimationStatus(c *gin.Context) {
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

	// 获取动画引擎
	animEngine := dev.GetAnimationEngine()

	// 获取已注册的动画列表
	availableAnimations := animEngine.GetRegisteredAnimations()

	// 获取当前状态
	isRunning := animEngine.IsRunning()
	currentName := animEngine.GetCurrentAnimation()

	response := AnimationStatusResponse{
		IsRunning:     isRunning,
		CurrentName:   currentName,
		AvailableList: availableAnimations,
	}

	c.JSON(http.StatusOK, ApiResponse{
		Status: "success",
		Data:   response,
	})
}
