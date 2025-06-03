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
