package legacy

import (
	"time"

	"hands/device"

	"github.com/gin-gonic/gin"
)

type LegacyServer struct {
	mapper    *InterfaceDeviceMapper
	startTime time.Time
}

// NewLegacyServer 创建新的兼容层 API 服务器实例
func NewLegacyServer(deviceManager *device.DeviceManager) (*LegacyServer, error) {
	mapper, err := NewInterfaceDeviceMapper(deviceManager)
	if err != nil {
		return nil, err
	}

	return &LegacyServer{
		mapper:    mapper,
		startTime: time.Now(),
	}, nil
}

// SetupRoutes 设置兼容层 API 路由
func (s *LegacyServer) SetupRoutes(r *gin.Engine) {
	// 兼容层 API 路由组
	legacy := r.Group("/api/legacy")
	{
		// 手型设置 API
		legacy.POST("/hand-type", s.handleHandType)

		// 手指姿态 API
		legacy.POST("/fingers", s.handleFingers)

		// 掌部姿态 API
		legacy.POST("/palm", s.handlePalm)

		// 预设姿势 API
		legacy.POST("/preset/:pose", s.handlePreset)

		// 动画控制 API
		legacy.POST("/animation", s.handleAnimation)

		// 获取传感器数据 API
		legacy.GET("/sensors", s.handleSensors)

		// 系统状态 API
		legacy.GET("/status", s.handleStatus)

		// 获取可用接口列表 API
		legacy.GET("/interfaces", s.handleInterfaces)

		// 获取手型配置 API
		legacy.GET("/hand-configs", s.handleHandConfigs)

		// 健康检查端点
		legacy.GET("/health", s.handleHealth)
	}
}
