package api2

import (
	"hands/device"
	"time"

	"github.com/gin-gonic/gin"
)

// Server API v2 服务器结构体
type Server struct {
	deviceManager *device.DeviceManager
	startTime     time.Time
	version       string
}

// NewServer 创建新的 API v2 服务器实例
func NewServer(deviceManager *device.DeviceManager) *Server {
	return &Server{
		deviceManager: deviceManager,
		startTime:     time.Now(),
		version:       "2.0.0",
	}
}

// SetupRoutes 设置 API v2 路由
func (s *Server) SetupRoutes(r *gin.Engine) {
	r.StaticFile("/", "./static/index.html")
	r.Static("/static", "./static")

	// API v2 路由组
	v2 := r.Group("/api/v2")
	{
		// 设备管理路由
		devices := v2.Group("/devices")
		{
			devices.GET("", s.handleGetDevices)                      // 获取所有设备列表
			devices.POST("", s.handleCreateDevice)                   // 创建新设备
			devices.GET("/:deviceId", s.handleGetDevice)             // 获取设备详情
			devices.DELETE("/:deviceId", s.handleDeleteDevice)       // 删除设备
			devices.PUT("/:deviceId/hand-type", s.handleSetHandType) // 设置手型

			// 设备级别的功能路由
			deviceRoutes := devices.Group("/:deviceId")
			{
				// 姿态控制路由
				poses := deviceRoutes.Group("/poses")
				{
					poses.POST("/fingers", s.handleSetFingerPose)      // 设置手指姿态
					poses.POST("/palm", s.handleSetPalmPose)           // 设置手掌姿态
					poses.POST("/preset/:pose", s.handleSetPresetPose) // 设置预设姿势
					poses.POST("/reset", s.handleResetPose)            // 重置姿态
				}

				// 动画控制路由
				animations := deviceRoutes.Group("/animations")
				{
					animations.GET("", s.handleGetAnimations)          // 获取可用动画列表
					animations.POST("/start", s.handleStartAnimation)  // 启动动画
					animations.POST("/stop", s.handleStopAnimation)    // 停止动画
					animations.GET("/status", s.handleAnimationStatus) // 获取动画状态
				}

				// 传感器数据路由
				sensors := deviceRoutes.Group("/sensors")
				{
					sensors.GET("", s.handleGetSensors)              // 获取所有传感器数据
					sensors.GET("/:sensorId", s.handleGetSensorData) // 获取特定传感器数据
				}

				// 设备状态路由
				deviceRoutes.GET("/status", s.handleGetDeviceStatus) // 获取设备状态
			}
		}

		// 系统管理路由
		system := v2.Group("/system")
		{
			system.GET("/models", s.handleGetSupportedModels) // 获取支持的设备型号
			system.GET("/status", s.handleGetSystemStatus)    // 获取系统状态
			system.GET("/health", s.handleHealthCheck)        // 健康检查
		}
	}
}
