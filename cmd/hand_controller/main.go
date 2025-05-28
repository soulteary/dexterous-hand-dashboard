package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"hands/pkg/config"
	"hands/pkg/control"
	"hands/pkg/control/modes"
	"hands/pkg/device"
	_ "hands/pkg/device/models" // 导入以注册设备类型
)

func main() {
	// 解析命令行参数
	configPath := flag.String("config", "config.json", "配置文件路径")
	flag.Parse()

	// 加载配置
	cfg, err := loadOrCreateConfig(*configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 创建设备管理器
	deviceManager := device.NewDeviceManager()

	// 创建并注册设备
	for _, deviceCfg := range cfg.Devices {
		dev, err := createDevice(deviceCfg, cfg.CanServiceURL)
		if err != nil {
			log.Printf("创建设备 %s 失败: %v", deviceCfg.ID, err)
			continue
		}

		if err := deviceManager.RegisterDevice(dev); err != nil {
			log.Printf("注册设备 %s 失败: %v", deviceCfg.ID, err)
			continue
		}

		log.Printf("成功注册设备: %s (%s)", deviceCfg.ID, deviceCfg.Model)
	}

	// 创建操作模式管理器
	modeManager := control.NewModeManager()

	// 注册操作模式
	modeManager.RegisterMode(modes.NewDirectPoseMode())

	animFactory := modes.NewAnimationFactory()
	for _, animType := range animFactory.GetSupportedAnimations() {
		anim, _ := animFactory.CreateAnimation(animType)
		modeManager.RegisterMode(anim)
	}

	// 设置默认设备
	if len(cfg.Devices) > 0 {
		defaultDev, err := deviceManager.GetDevice(cfg.DefaultDevice.ID)
		if err != nil {
			log.Printf("获取默认设备失败: %v", err)
		} else {
			modeManager.SetDevice(defaultDev)

			// 连接设备
			if err := defaultDev.Connect(); err != nil {
				log.Printf("连接设备失败: %v", err)
			} else {
				log.Printf("成功连接设备: %s", cfg.DefaultDevice.ID)
			}
		}
	}

	// 启动默认模式
	if err := modeManager.SwitchMode("DirectPose", nil); err != nil {
		log.Printf("启动默认模式失败: %v", err)
	} else {
		log.Println("成功启动直接姿态控制模式")
	}

	// 演示功能
	demonstrateFeatures(modeManager, deviceManager)

	// 等待中断信号
	waitForShutdown()

	log.Println("应用程序正在关闭...")
}

func loadOrCreateConfig(configPath string) (*config.Config, error) {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// 配置文件不存在，创建默认配置
		cfg := config.GetDefaultConfig()
		if err := config.SaveConfig(cfg, configPath); err != nil {
			return nil, fmt.Errorf("保存默认配置失败：%w", err)
		}
		log.Printf("创建默认配置文件: %s", configPath)
		return cfg, nil
	}

	return config.LoadConfig(configPath)
}

func createDevice(deviceCfg config.DeviceConfig, canServiceURL string) (device.Device, error) {
	deviceConfig := map[string]interface{}{
		"id":              deviceCfg.ID,
		"can_service_url": canServiceURL,
		"can_interface":   deviceCfg.CanInterface,
	}

	// 合并设备特定参数
	for k, v := range deviceCfg.Parameters {
		deviceConfig[k] = v
	}

	return device.CreateDevice(deviceCfg.Model, deviceConfig)
}

func demonstrateFeatures(modeManager *control.ModeManager, deviceManager *device.DeviceManager) {
	log.Println("开始功能演示...")

	// 演示直接姿态控制
	log.Println("演示直接姿态控制...")
	if activeMode := modeManager.GetActiveMode(); activeMode != nil {
		if poseMode, ok := activeMode.(*modes.DirectPoseMode); ok {
			// 获取设备
			devices := deviceManager.GetAllDevices()
			if len(devices) > 0 {
				dev := devices[0]

				// 发送手指姿态
				fingerPoses := map[string][]byte{
					"thumb":  {45},
					"index":  {30},
					"middle": {60},
					"ring":   {45},
					"pinky":  {30},
				}

				inputs := map[string]interface{}{
					"finger_poses": fingerPoses,
				}

				if err := poseMode.Update(dev, inputs); err != nil {
					log.Printf("发送手指姿态失败: %v", err)
				} else {
					log.Println("成功发送手指姿态")
				}

				time.Sleep(2 * time.Second)
			}
		}
	}

	// 演示动画模式
	log.Println("演示波浪动画...")
	params := map[string]interface{}{
		"duration": 3 * time.Second,
		"frames":   30,
	}

	if err := modeManager.SwitchMode("Animation_wave", params); err != nil {
		log.Printf("切换到波浪动画失败: %v", err)
	} else {
		log.Println("成功启动波浪动画")
		time.Sleep(4 * time.Second)
	}

	// 切换回直接控制模式
	if err := modeManager.SwitchMode("DirectPose", nil); err != nil {
		log.Printf("切换回直接控制模式失败: %v", err)
	}
}

func waitForShutdown() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
}
