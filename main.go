package main

import (
	"fmt"
	"hands/api"
	"hands/cli"
	"hands/config"
	"log"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// 初始化服务
func initService() {
	log.Printf("🔧 服务配置：")
	log.Printf("   - CAN 服务 URL: %s", config.Config.CanServiceURL)
	log.Printf("   - Web 端口: %s", config.Config.WebPort)
	log.Printf("   - 可用接口: %v", config.Config.AvailableInterfaces)
	log.Printf("   - 默认接口: %s", config.Config.DefaultInterface)

	log.Println("✅ 控制服务初始化完成")
}

func printUsage() {
	fmt.Println("CAN Control Service with Hand Type Support")
	fmt.Println("Usage:")
	fmt.Println("  -can-url string         CAN 服务的 URL (default: http://127.0.0.1:5260)")
	fmt.Println("  -port string            Web 服务的端口 (default: 9099)")
	fmt.Println("  -interface string       默认 CAN 接口")
	fmt.Println("  -can-interfaces string  支持的 CAN 接口列表，用逗号分隔")
	fmt.Println("")
	fmt.Println("Environment Variables:")
	fmt.Println("  CAN_SERVICE_URL        CAN 服务的 URL")
	fmt.Println("  WEB_PORT              Web 服务的端口")
	fmt.Println("  DEFAULT_INTERFACE     默认 CAN 接口")
	fmt.Println("  CAN_INTERFACES        支持的 CAN 接口列表，用逗号分隔")
	fmt.Println("")
	fmt.Println("New Features:")
	fmt.Println("  - Support for left/right hand configuration")
	fmt.Println("  - Dynamic CAN ID assignment based on hand type")
	fmt.Println("  - Hand type API endpoints")
	fmt.Println("  - Enhanced logging with hand type information")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  ./control-service -can-interfaces can0,can1,vcan0")
	fmt.Println("  ./control-service -interface can1 -can-interfaces can0,can1")
	fmt.Println("  CAN_INTERFACES=can0,can1,vcan0 ./control-service")
	fmt.Println("  CAN_SERVICE_URL=http://localhost:5260 ./control-service")
}

func main() {
	// 检查是否请求帮助
	if len(os.Args) > 1 && (os.Args[1] == "-h" || os.Args[1] == "--help") {
		printUsage()
		return
	}

	// 解析配置
	config.Config = cli.ParseConfig()

	// 验证配置
	if len(config.Config.AvailableInterfaces) == 0 {
		log.Fatal("❌ 没有可用的 CAN 接口")
	}

	if config.Config.DefaultInterface == "" {
		log.Fatal("❌ 没有设置默认 CAN 接口")
	}

	// 记录启动时间
	api.ServerStartTime = time.Now()

	log.Printf("🚀 启动 CAN 控制服务 (支持左右手配置)")

	// 初始化服务
	initService()

	// 设置 Gin 模式
	gin.SetMode(gin.ReleaseMode)

	// 创建 Gin 引擎
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"}, // 允许的域，*表示允许所有
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	models.RegisterDeviceTypes()

	// 设置 API 路由
	api2.NewServer(device.NewDeviceManager()).SetupRoutes(r)

	// 启动服务器
	log.Printf("🌐 CAN 控制服务运行在 http://localhost:%s", config.Config.WebPort)
	log.Printf("📡 连接到 CAN 服务: %s", config.Config.CanServiceURL)
	log.Printf("🎯 默认接口: %s", config.Config.DefaultInterface)
	log.Printf("🔌 可用接口: %v", config.Config.AvailableInterfaces)
	log.Printf("🤖 支持左右手动态配置")

	if err := r.Run(":" + config.Config.WebPort); err != nil {
		log.Fatalf("❌ 服务启动失败: %v", err)
	}
}
