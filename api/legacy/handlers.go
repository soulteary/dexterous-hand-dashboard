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
