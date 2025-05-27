package api

import (
	"time"

	"github.com/gin-gonic/gin"
)

// 全局变量
var (
	ServerStartTime time.Time
)

func SetupRoutes(r *gin.Engine) {
	r.StaticFile("/", "./static/index.html")
	r.Static("/static", "./static")

	api := r.Group("/api")
	{
		// 手型设置 API
		api.POST("/hand-type", HandleHandType)

		// 手指姿态 API
		api.POST("/fingers", HandleFingers)

		// 掌部姿态 API
		api.POST("/palm", HandlePalm)

		// 预设姿势 API
		api.POST("/preset/:pose", HandlePreset)

		// 动画控制 API
		api.POST("/animation", HandleAnimation)

		// 获取传感器数据 API
		api.GET("/sensors", HandleSensors)

		// 系统状态 API
		api.GET("/status", HandleStatus)

		// 获取可用接口列表 API
		api.GET("/interfaces", HandleInterfaces)

		// 获取手型配置 API
		api.GET("/hand-configs", HandleHandConfigs)

		// 健康检查端点
		api.GET("/health", HandleHealth)
	}
}
