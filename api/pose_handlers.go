package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// handleSetFingerPose 设置手指姿态
func (s *Server) handleSetFingerPose(c *gin.Context) {
	deviceId := c.Param("deviceId")

	var req FingerPoseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ApiResponse{
			Status: "error",
			Error:  "无效的手指姿态数据：" + err.Error(),
		})
		return
	}

	// 验证每个值是否在范围内
	for _, v := range req.Pose {
		if v > 255 {
			c.JSON(http.StatusBadRequest, ApiResponse{
				Status: "error",
				Error:  "手指姿态值必须在 0-255 范围内",
			})
			return
		}
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

	// 停止当前动画（如果正在运行）
	animEngine := dev.GetAnimationEngine()
	if animEngine.IsRunning() {
		if err := animEngine.Stop(); err != nil {
			c.JSON(http.StatusInternalServerError, ApiResponse{
				Status: "error",
				Error:  fmt.Sprintf("停止动画失败：%v", err),
			})
			return
		}
	}

	// 设置手指姿态
	if err := dev.SetFingerPose(req.Pose); err != nil {
		c.JSON(http.StatusInternalServerError, ApiResponse{
			Status: "error",
			Error:  "发送手指姿态失败：" + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, ApiResponse{
		Status:  "success",
		Message: "手指姿态指令发送成功",
		Data: map[string]any{
			"deviceId": deviceId,
			"pose":     req.Pose,
		},
	})
}

// handleSetPalmPose 设置手掌姿态
func (s *Server) handleSetPalmPose(c *gin.Context) {
	deviceId := c.Param("deviceId")

	var req PalmPoseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ApiResponse{
			Status: "error",
			Error:  "无效的掌部姿态数据：" + err.Error(),
		})
		return
	}

	// 验证每个值是否在范围内
	for _, v := range req.Pose {
		if v > 255 {
			c.JSON(http.StatusBadRequest, ApiResponse{
				Status: "error",
				Error:  "掌部姿态值必须在 0-255 范围内",
			})
			return
		}
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

	// 停止当前动画（如果正在运行）
	animEngine := dev.GetAnimationEngine()
	if animEngine.IsRunning() {
		if err := animEngine.Stop(); err != nil {
			c.JSON(http.StatusInternalServerError, ApiResponse{
				Status: "error",
				Error:  fmt.Sprintf("停止动画失败：%v", err),
			})
			return
		}
	}

	// 设置手掌姿态
	if err := dev.SetPalmPose(req.Pose); err != nil {
		c.JSON(http.StatusInternalServerError, ApiResponse{
			Status: "error",
			Error:  "发送掌部姿态失败：" + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, ApiResponse{
		Status:  "success",
		Message: "掌部姿态指令发送成功",
		Data: map[string]any{
			"deviceId": deviceId,
			"pose":     req.Pose,
		},
	})
}

// handleSetPresetPose 设置预设姿势
func (s *Server) handleSetPresetPose(c *gin.Context) {
	deviceId := c.Param("deviceId")
	pose := c.Param("pose")

	// 获取设备
	dev, err := s.deviceManager.GetDevice(deviceId)
	if err != nil {
		c.JSON(http.StatusNotFound, ApiResponse{
			Status: "error",
			Error:  fmt.Sprintf("设备 %s 不存在", deviceId),
		})
		return
	}

	// 停止当前动画（如果正在运行）
	animEngine := dev.GetAnimationEngine()
	if animEngine.IsRunning() {
		if err := animEngine.Stop(); err != nil {
			c.JSON(http.StatusInternalServerError, ApiResponse{
				Status: "error",
				Error:  fmt.Sprintf("停止动画失败：%v", err),
			})
			return
		}
	}

	// 使用设备的预设姿势方法
	if err := dev.ExecutePreset(pose); err != nil {
		c.JSON(http.StatusBadRequest, ApiResponse{
			Status: "error",
			Error:  fmt.Sprintf("执行预设姿势失败: %v", err),
		})
		return
	}

	// 获取预设姿势的描述
	description := dev.GetPresetDescription(pose)
	message := fmt.Sprintf("已设置预设姿势: %s", pose)
	if description != "" {
		message = fmt.Sprintf("已设置%s", description)
	}

	c.JSON(http.StatusOK, ApiResponse{
		Status:  "success",
		Message: message,
		Data: map[string]any{
			"deviceId":    deviceId,
			"pose":        pose,
			"description": description,
		},
	})
}

// handleResetPose 重置姿态
func (s *Server) handleResetPose(c *gin.Context) {
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

	// 停止当前动画（如果正在运行）
	animEngine := dev.GetAnimationEngine()
	if animEngine.IsRunning() {
		if err := animEngine.Stop(); err != nil {
			c.JSON(http.StatusInternalServerError, ApiResponse{
				Status: "error",
				Error:  fmt.Sprintf("停止动画失败：%v", err),
			})
			return
		}
	}

	// 重置姿态
	if err := dev.ResetPose(); err != nil {
		c.JSON(http.StatusInternalServerError, ApiResponse{
			Status: "error",
			Error:  "重置姿态失败：" + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, ApiResponse{
		Status:  "success",
		Message: fmt.Sprintf("设备 %s 已重置到默认姿态", deviceId),
		Data: map[string]any{
			"deviceId": deviceId,
		},
	})
}

// handleGetPresetPose 获取设备支持的预设姿势列表
func (s *Server) handleGetPresetPose(c *gin.Context) {
	deviceID := c.Param("deviceId")

	device, err := s.deviceManager.GetDevice(deviceID)
	if err != nil {
		c.JSON(http.StatusNotFound, ApiResponse{
			Status: "error",
			Error:  fmt.Sprintf("设备 %s 不存在", deviceID),
		})
		return
	}

	// 使用设备的预设姿势方法
	presets := device.GetSupportedPresets()

	// 构建详细的预设信息
	presetDetails := make([]map[string]string, 0, len(presets))
	for _, presetName := range presets {
		description := device.GetPresetDescription(presetName)
		presetDetails = append(presetDetails, map[string]string{
			"name":        presetName,
			"description": description,
		})
	}

	c.JSON(http.StatusOK, ApiResponse{
		Status:  "success",
		Message: "获取设备支持的预设姿势列表成功",
		Data: map[string]any{
			"deviceId": deviceID,
			"presets":  presetDetails,
			"count":    len(presets),
		},
	})
}
