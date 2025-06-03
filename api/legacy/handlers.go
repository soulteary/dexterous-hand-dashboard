package legacy

import (
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
