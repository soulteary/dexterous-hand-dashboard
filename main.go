package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// 手型类型常量
const HAND_TYPE_LEFT = 0x28
const HAND_TYPE_RIGHT = 0x27

// O7_MODIFIED: 设备型号常量
const (
	DEVICE_TYPE_L10 = "L10" // L10型号
	DEVICE_TYPE_O7  = "O7"  // O7型号
)

// O7_MODIFIED: 设备型号配置
var (
	// 全局默认设备型号，可通过命令行参数修改
	defaultDeviceType = DEVICE_TYPE_L10
)

// API 请求结构体 - 添加手型支持
type FingerPoseRequest struct {
	Interface  string `json:"interface,omitempty"`
	Pose       []byte `json:"pose" binding:"required"`
	HandType   string `json:"handType,omitempty"`   // 手型类型
	HandId     uint32 `json:"handId,omitempty"`     // CAN ID
	DeviceType string `json:"deviceType,omitempty"` // O7_MODIFIED: 设备型号
}

type PalmPoseRequest struct {
	Interface  string `json:"interface,omitempty"`
	Pose       []byte `json:"pose" binding:"required,len=4"`
	HandType   string `json:"handType,omitempty"`   // 手型类型
	HandId     uint32 `json:"handId,omitempty"`     // CAN ID
	DeviceType string `json:"deviceType,omitempty"` // O7_MODIFIED: 设备型号
}

type AnimationRequest struct {
	Interface  string `json:"interface,omitempty"`
	Type       string `json:"type" binding:"required,oneof=wave sway stop"`
	Speed      int    `json:"speed" binding:"min=0,max=2000"`
	HandType   string `json:"handType,omitempty"`   // 手型类型
	HandId     uint32 `json:"handId,omitempty"`     // CAN ID
	DeviceType string `json:"deviceType,omitempty"` // O7_MODIFIED: 设备型号
}

// O7_MODIFIED: 速度设置请求
type SpeedRequest struct {
	Interface  string `json:"interface,omitempty"`
	Speeds     []byte `json:"speeds" binding:"required"`
	HandType   string `json:"handType,omitempty"`   // 手型类型
	HandId     uint32 `json:"handId,omitempty"`     // CAN ID
	DeviceType string `json:"deviceType,omitempty"` // 设备型号
}

// 手型设置请求
type HandTypeRequest struct {
	Interface  string `json:"interface" binding:"required"`
	HandType   string `json:"handType" binding:"required,oneof=left right"`
	HandId     uint32 `json:"handId" binding:"required"`
	DeviceType string `json:"deviceType,omitempty"` // O7_MODIFIED: 设备型号
}

// CAN 服务请求结构体
type CanMessage struct {
	Interface string `json:"interface"`
	ID        uint32 `json:"id"`
	Data      []byte `json:"data"`
}

// 传感器数据结构体
type SensorData struct {
	Interface    string    `json:"interface"`
	Thumb        int       `json:"thumb"`
	Index        int       `json:"index"`
	Middle       int       `json:"middle"`
	Ring         int       `json:"ring"`
	Pinky        int       `json:"pinky"`
	PalmPosition []byte    `json:"palmPosition"`
	LastUpdate   time.Time `json:"lastUpdate"`
	DeviceType   string    `json:"deviceType,omitempty"` // O7_MODIFIED: 设备型号
}

// API 响应结构体
type ApiResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message,omitempty"`
	Error   string      `json:"error,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// 配置结构体
type Config struct {
	CanServiceURL       string
	WebPort             string
	DefaultInterface    string
	AvailableInterfaces []string
	DeviceType          string // O7_MODIFIED: 设备型号配置
}

// 手型配置结构体
type HandConfig struct {
	HandType   string `json:"handType"`
	HandId     uint32 `json:"handId"`
	DeviceType string `json:"deviceType"` // O7_MODIFIED: 设备型号
}

// 全局变量
var (
	sensorDataMap    map[string]*SensorData // 每个接口的传感器数据
	sensorMutex      sync.RWMutex
	animationActive  map[string]bool // 每个接口的动画状态
	animationMutex   sync.Mutex
	stopAnimationMap map[string]chan struct{} // 每个接口的停止动画通道
	handConfigs      map[string]*HandConfig   // 每个接口的手型配置
	handConfigMutex  sync.RWMutex
	config           *Config
	serverStartTime  time.Time
)

// 解析配置
func parseConfig() *Config {
	cfg := &Config{}

	// 命令行参数
	var canInterfacesFlag string
	flag.StringVar(&cfg.CanServiceURL, "can-url", "http://192.168.128.35:5260", "CAN 服务的 URL")
	flag.StringVar(&cfg.WebPort, "port", "9099", "Web 服务的端口")
	flag.StringVar(&cfg.DefaultInterface, "interface", "", "默认 CAN 接口")
	flag.StringVar(&canInterfacesFlag, "can-interfaces", "", "支持的 CAN 接口列表，用逗号分隔 (例如: can0,can1,vcan0)")
	// O7_MODIFIED: 添加设备类型参数
	flag.StringVar(&cfg.DeviceType, "device-type", defaultDeviceType, "设备类型 (L10 或 O7)")
	flag.Parse()

	// 环境变量覆盖命令行参数
	if envURL := os.Getenv("CAN_SERVICE_URL"); envURL != "" {
		cfg.CanServiceURL = envURL
	}
	if envPort := os.Getenv("WEB_PORT"); envPort != "" {
		cfg.WebPort = envPort
	}
	if envInterface := os.Getenv("DEFAULT_INTERFACE"); envInterface != "" {
		cfg.DefaultInterface = envInterface
	}
	if envInterfaces := os.Getenv("CAN_INTERFACES"); envInterfaces != "" {
		canInterfacesFlag = envInterfaces
	}
	// O7_MODIFIED: 添加设备类型环境变量
	if envDeviceType := os.Getenv("DEVICE_TYPE"); envDeviceType != "" {
		cfg.DeviceType = envDeviceType
	}

	// 验证设备类型
	if cfg.DeviceType != DEVICE_TYPE_L10 && cfg.DeviceType != DEVICE_TYPE_O7 {
		log.Printf("⚠️ 无效的设备类型: %s，使用默认值: %s", cfg.DeviceType, defaultDeviceType)
		cfg.DeviceType = defaultDeviceType
	}

	// 解析可用接口
	if canInterfacesFlag != "" {
		cfg.AvailableInterfaces = strings.Split(canInterfacesFlag, ",")
		// 清理空白字符
		for i, iface := range cfg.AvailableInterfaces {
			cfg.AvailableInterfaces[i] = strings.TrimSpace(iface)
		}
	}

	// 如果没有指定可用接口，从CAN服务获取
	if len(cfg.AvailableInterfaces) == 0 {
		log.Println("🔍 未指定可用接口，将从 CAN 服务获取...")
		cfg.AvailableInterfaces = getAvailableInterfacesFromCanService(cfg.CanServiceURL)
	}

	// 设置默认接口
	if cfg.DefaultInterface == "" && len(cfg.AvailableInterfaces) > 0 {
		cfg.DefaultInterface = cfg.AvailableInterfaces[0]
	}

	return cfg
}

// 从CAN服务获取可用接口
func getAvailableInterfacesFromCanService(canServiceURL string) []string {
	resp, err := http.Get(canServiceURL + "/api/interfaces")
	if err != nil {
		log.Printf("⚠️ 无法从 CAN 服务获取接口列表: %v，使用默认配置", err)
		return []string{"can0", "can1"} // 默认接口
	}
	defer resp.Body.Close()

	var apiResp ApiResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		log.Printf("⚠️ 解析 CAN 服务接口响应失败: %v，使用默认配置", err)
		return []string{"can0", "can1"}
	}

	if data, ok := apiResp.Data.(map[string]interface{}); ok {
		if configuredPorts, ok := data["configuredPorts"].([]interface{}); ok {
			interfaces := make([]string, 0, len(configuredPorts))
			for _, port := range configuredPorts {
				if portStr, ok := port.(string); ok {
					interfaces = append(interfaces, portStr)
				}
			}
			if len(interfaces) > 0 {
				log.Printf("✅ 从 CAN 服务获取到接口: %v", interfaces)
				return interfaces
			}
		}
	}

	log.Println("⚠️ 无法从 CAN 服务获取有效接口，使用默认配置")
	return []string{"can0", "can1"}
}

// 验证接口是否可用
func isValidInterface(ifName string) bool {
	for _, validIface := range config.AvailableInterfaces {
		if ifName == validIface {
			return true
		}
	}
	return false
}

// 获取或创建手型配置
func getHandConfig(ifName string) *HandConfig {
	handConfigMutex.RLock()
	if handConfig, exists := handConfigs[ifName]; exists {
		handConfigMutex.RUnlock()
		return handConfig
	}
	handConfigMutex.RUnlock()

	// 创建默认配置
	handConfigMutex.Lock()
	defer handConfigMutex.Unlock()

	// 再次检查（双重检查锁定）
	if handConfig, exists := handConfigs[ifName]; exists {
		return handConfig
	}

	// 创建默认配置（右手，设备类型继承全局配置）
	handConfigs[ifName] = &HandConfig{
		HandType:   "right",
		HandId:     HAND_TYPE_RIGHT,
		DeviceType: config.DeviceType, // O7_MODIFIED: 使用全局设备类型
	}

	log.Printf("🆕 为接口 %s 创建默认手型配置: 右手 (0x%X), 设备类型: %s",
		ifName, HAND_TYPE_RIGHT, config.DeviceType)
	return handConfigs[ifName]
}

// 设置手型配置
func setHandConfig(ifName, handType string, handId uint32) {
	handConfigMutex.Lock()
	defer handConfigMutex.Unlock()

	// 获取现有配置或使用默认设备类型
	deviceType := config.DeviceType
	if existing, exists := handConfigs[ifName]; exists {
		deviceType = existing.DeviceType
	}

	handConfigs[ifName] = &HandConfig{
		HandType:   handType,
		HandId:     handId,
		DeviceType: deviceType, // O7_MODIFIED: 保持现有设备类型或使用默认值
	}

	log.Printf("🔧 接口 %s 手型配置已更新: %s (0x%X), 设备类型: %s",
		ifName, handType, handId, deviceType)
}

// 解析手型参数
func parseHandType(handType string, handId uint32, ifName string) uint32 {
	// 如果提供了有效的handId，直接使用
	if handId != 0 {
		return handId
	}

	// 根据handType字符串确定ID
	switch strings.ToLower(handType) {
	case "left":
		return HAND_TYPE_LEFT
	case "right":
		return HAND_TYPE_RIGHT
	default:
		// 使用接口的配置
		handConfig := getHandConfig(ifName)
		return handConfig.HandId
	}
}

// 初始化服务
func initService() {
	log.Printf("🔧 服务配置:")
	log.Printf("   - CAN 服务 URL: %s", config.CanServiceURL)
	log.Printf("   - Web 端口: %s", config.WebPort)
	log.Printf("   - 可用接口: %v", config.AvailableInterfaces)
	log.Printf("   - 默认接口: %s", config.DefaultInterface)
	log.Printf("   - 设备类型: %s", config.DeviceType) // O7_MODIFIED: 显示设备类型

	// 初始化传感器数据映射
	sensorDataMap = make(map[string]*SensorData)
	for _, ifName := range config.AvailableInterfaces {
		sensorDataMap[ifName] = &SensorData{
			Interface:    ifName,
			Thumb:        0,
			Index:        0,
			Middle:       0,
			Ring:         0,
			Pinky:        0,
			PalmPosition: []byte{128, 128, 128, 128},
			LastUpdate:   time.Now(),
			DeviceType:   config.DeviceType, // O7_MODIFIED: 设置设备类型
		}
	}

	// 初始化动画状态映射
	animationActive = make(map[string]bool)
	stopAnimationMap = make(map[string]chan struct{})
	for _, ifName := range config.AvailableInterfaces {
		animationActive[ifName] = false
		stopAnimationMap[ifName] = make(chan struct{}, 1)
	}

	// 初始化手型配置映射
	handConfigs = make(map[string]*HandConfig)

	log.Println("✅ 控制服务初始化完成")
}

// 发送请求到 CAN 服务
func sendToCanService(msg CanMessage) error {
	jsonData, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("JSON 编码错误: %v", err)
	}

	resp, err := http.Post(config.CanServiceURL+"/api/can", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("CAN 服务请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp ApiResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			return fmt.Errorf("CAN 服务返回错误: HTTP %d", resp.StatusCode)
		}
		return fmt.Errorf("CAN 服务返回错误: %s", errResp.Error)
	}

	return nil
}

// O7_MODIFIED: 创建适合不同设备类型的CAN消息
func createFingerPoseMessage(handConfig *HandConfig, ifName string, pose []byte, canId uint32) CanMessage {
	var data []byte

	// 根据设备类型构造不同格式的数据
	if handConfig.DeviceType == DEVICE_TYPE_O7 {
		// O7型号: 0x01指令后跟7个关节位置
		if len(pose) == 6 {
			// 如果传入的是6个值(L10格式)，我们需要添加第7个值
			// 在这里默认将第7个关节设为中间值
			data = append([]byte{0x01}, pose...)
			data = append(data, 128) // 添加第7个关节值为中间值
		} else {
			// 假设传入的已经是7个值
			data = append([]byte{0x01}, pose...)
		}
	} else {
		// L10型号: 0x01指令后跟6个值
		data = append([]byte{0x01}, pose...)
	}

	return CanMessage{
		Interface: ifName,
		ID:        canId,
		Data:      data,
	}
}

// 发送手指姿态指令 - 支持设备类型
func sendFingerPose(ifName string, pose []byte, handType string, handId uint32) error {
	// O7_MODIFIED: 根据设备类型验证数据长度
	deviceType := config.DeviceType
	if handConfig, exists := handConfigs[ifName]; exists {
		deviceType = handConfig.DeviceType
	}

	// 验证姿态数据长度
	if deviceType == DEVICE_TYPE_O7 {
		if len(pose) != 7 && len(pose) != 6 {
			return fmt.Errorf("O7设备需要7个关节值(或兼容L10的6个值)")
		}
	} else {
		if len(pose) != 6 {
			return fmt.Errorf("L10设备需要6个关节值")
		}
	}

	// 如果未指定接口，使用默认接口
	if ifName == "" {
		ifName = config.DefaultInterface
	}

	// 验证接口
	if !isValidInterface(ifName) {
		return fmt.Errorf("无效的接口 %s，可用接口: %v", ifName, config.AvailableInterfaces)
	}

	// 解析手型ID
	canId := parseHandType(handType, handId, ifName)

	// 添加随机扰动
	perturbedPose := make([]byte, len(pose))
	for i, v := range pose {
		perturbedPose[i] = perturb(v, 5)
	}

	// 获取当前接口的手型配置
	handConfig := getHandConfig(ifName)
	// 确保使用正确的接口名称

	// O7_MODIFIED: 构造适合设备类型的CAN消息
	msg := createFingerPoseMessage(handConfig, ifName, perturbedPose, canId)

	err := sendToCanService(msg)
	if err == nil {
		handTypeName := "右手"
		if canId == HAND_TYPE_LEFT {
			handTypeName = "左手"
		}

		// O7_MODIFIED: 根据设备类型打印不同的日志
		if handConfig.DeviceType == DEVICE_TYPE_O7 {
			if len(perturbedPose) == 7 {
				log.Printf("✅ %s (%s, 0x%X, %s) 手指动作已发送: [%X %X %X %X %X %X %X]",
					ifName, handTypeName, canId, handConfig.DeviceType,
					perturbedPose[0], perturbedPose[1], perturbedPose[2],
					perturbedPose[3], perturbedPose[4], perturbedPose[5], perturbedPose[6])
			} else {
				// 6个值情况下，补充日志中的第7个值
				log.Printf("✅ %s (%s, 0x%X, %s) 手指动作已发送: [%X %X %X %X %X %X +128]",
					ifName, handTypeName, canId, handConfig.DeviceType,
					perturbedPose[0], perturbedPose[1], perturbedPose[2],
					perturbedPose[3], perturbedPose[4], perturbedPose[5])
			}
		} else {
			log.Printf("✅ %s (%s, 0x%X, %s) 手指动作已发送: [%X %X %X %X %X %X]",
				ifName, handTypeName, canId, handConfig.DeviceType,
				perturbedPose[0], perturbedPose[1], perturbedPose[2],
				perturbedPose[3], perturbedPose[4], perturbedPose[5])
		}
	} else {
		log.Printf("❌ %s 手指控制发送失败: %v", ifName, err)
	}

	return err
}

// O7_MODIFIED: 创建适合不同设备类型的掌部姿态消息
func createPalmPoseMessage(handConfig *HandConfig, ifName string, pose []byte) CanMessage {
	var data []byte

	// 根据设备类型构造不同格式的数据
	if handConfig.DeviceType == DEVICE_TYPE_O7 {
		// O7型号：不使用0x04指令，而是扩展0x01指令的数据
		// 这里我们需要获取之前的关节位置来构造完整的7个关节
		// 默认使用中间值
		fingerPose := []byte{128, 128, 128, 128, 128, 128, 128}

		// 如果提供了掌部数据，将后4个替换为提供的值
		if len(pose) == 4 {
			// 将提供的掌部数据映射到关节4-7
			// 这只是一个示例映射方式，实际映射需要根据具体硬件定义
			fingerPose[3] = pose[0]
			fingerPose[4] = pose[1]
			fingerPose[5] = pose[2]
			fingerPose[6] = pose[3]
		}

		// 添加指令码
		data = append([]byte{0x01}, fingerPose...)
	} else {
		// L10型号: 0x04指令后跟4个掌部关节值
		data = append([]byte{0x04}, pose...)
	}

	return CanMessage{
		Interface: ifName,
		ID:        handConfig.HandId,
		Data:      data,
	}
}

// 发送掌部姿态指令 - 支持设备类型
func sendPalmPose(ifName string, pose []byte, handType string, handId uint32) error {
	if len(pose) != 4 {
		return fmt.Errorf("无效的姿态数据长度，需要 4 个字节")
	}

	// 如果未指定接口，使用默认接口
	if ifName == "" {
		ifName = config.DefaultInterface
	}

	// 验证接口
	if !isValidInterface(ifName) {
		return fmt.Errorf("无效的接口 %s，可用接口: %v", ifName, config.AvailableInterfaces)
	}

	// 解析手型ID
	canId := parseHandType(handType, handId, ifName)

	// 添加随机扰动
	perturbedPose := make([]byte, len(pose))
	for i, v := range pose {
		perturbedPose[i] = perturb(v, 8)
	}

	// 获取当前接口的手型配置
	handConfig := getHandConfig(ifName)

	// O7_MODIFIED: 构造适合设备类型的CAN消息
	msg := createPalmPoseMessage(handConfig, ifName, perturbedPose)

	err := sendToCanService(msg)
	if err == nil {
		handTypeName := "右手"
		if canId == HAND_TYPE_LEFT {
			handTypeName = "左手"
		}

		// O7_MODIFIED: 根据设备类型显示不同的日志
		if handConfig.DeviceType == DEVICE_TYPE_O7 {
			log.Printf("✅ %s (%s, 0x%X, %s) 掌部姿态已发送（通过0x01指令）: [%X %X %X %X]",
				ifName, handTypeName, canId, handConfig.DeviceType,
				perturbedPose[0], perturbedPose[1], perturbedPose[2], perturbedPose[3])
		} else {
			log.Printf("✅ %s (%s, 0x%X, %s) 掌部姿态已发送: [%X %X %X %X]",
				ifName, handTypeName, canId, handConfig.DeviceType,
				perturbedPose[0], perturbedPose[1], perturbedPose[2], perturbedPose[3])
		}

		// 更新传感器数据中的掌部位置
		sensorMutex.Lock()
		if sensorData, exists := sensorDataMap[ifName]; exists {
			copy(sensorData.PalmPosition, perturbedPose)
			sensorData.LastUpdate = time.Now()
		}
		sensorMutex.Unlock()
	} else {
		log.Printf("❌ %s 掌部控制发送失败: %v", ifName, err)
	}

	return err
}

// O7_MODIFIED: 为O7设备发送关节速度
func sendJointSpeeds(ifName string, speeds []byte, handType string, handId uint32) error {
	// 验证速度数据长度
	deviceType := config.DeviceType
	if handConfig, exists := handConfigs[ifName]; exists {
		deviceType = handConfig.DeviceType
	}

	// 验证速度数据长度
	if deviceType == DEVICE_TYPE_O7 {
		if len(speeds) != 7 && len(speeds) != 6 {
			return fmt.Errorf("O7设备需要7个关节速度值(或兼容L10的6个值)")
		}
	} else {
		if len(speeds) != 5 {
			return fmt.Errorf("L10设备需要5个手指的速度值")
		}
	}

	// 如果未指定接口，使用默认接口
	if ifName == "" {
		ifName = config.DefaultInterface
	}

	// 验证接口
	if !isValidInterface(ifName) {
		return fmt.Errorf("无效的接口 %s，可用接口: %v", ifName, config.AvailableInterfaces)
	}

	// 解析手型ID
	canId := parseHandType(handType, handId, ifName)

	// 添加随机扰动
	perturbedSpeeds := make([]byte, len(speeds))
	for i, v := range speeds {
		perturbedSpeeds[i] = perturb(v, 3) // 速度值扰动较小
	}

	// 获取当前接口的手型配置
	handConfig := getHandConfig(ifName)

	// 构造CAN消息
	var data []byte
	if handConfig.DeviceType == DEVICE_TYPE_O7 {
		// O7型号: 0x05指令后跟7个关节速度
		if len(perturbedSpeeds) == 6 {
			// 如果传入的是6个值(L10格式)，我们需要添加第7个值
			data = append([]byte{0x05}, perturbedSpeeds...)
			data = append(data, perturbedSpeeds[5]) // 复制最后一个值作为第7个关节速度
		} else {
			data = append([]byte{0x05}, perturbedSpeeds...)
		}
	} else {
		// L10型号: 0x05指令后跟5个手指速度
		data = append([]byte{0x05}, perturbedSpeeds...)
	}

	msg := CanMessage{
		Interface: ifName,
		ID:        canId,
		Data:      data,
	}

	err := sendToCanService(msg)
	if err == nil {
		handTypeName := "右手"
		if canId == HAND_TYPE_LEFT {
			handTypeName = "左手"
		}

		if handConfig.DeviceType == DEVICE_TYPE_O7 {
			if len(perturbedSpeeds) == 7 {
				log.Printf("✅ %s (%s, 0x%X, %s) 关节速度已发送: [%X %X %X %X %X %X %X]",
					ifName, handTypeName, canId, handConfig.DeviceType,
					perturbedSpeeds[0], perturbedSpeeds[1], perturbedSpeeds[2],
					perturbedSpeeds[3], perturbedSpeeds[4], perturbedSpeeds[5], perturbedSpeeds[6])
			} else {
				// 6个值情况，日志中补充第7个值
				log.Printf("✅ %s (%s, 0x%X, %s) 关节速度已发送: [%X %X %X %X %X %X +%X]",
					ifName, handTypeName, canId, handConfig.DeviceType,
					perturbedSpeeds[0], perturbedSpeeds[1], perturbedSpeeds[2],
					perturbedSpeeds[3], perturbedSpeeds[4], perturbedSpeeds[5], perturbedSpeeds[5])
			}
		} else {
			log.Printf("✅ %s (%s, 0x%X, %s) 手指速度已发送: [%X %X %X %X %X]",
				ifName, handTypeName, canId, handConfig.DeviceType,
				perturbedSpeeds[0], perturbedSpeeds[1], perturbedSpeeds[2],
				perturbedSpeeds[3], perturbedSpeeds[4])
		}
	} else {
		log.Printf("❌ %s 速度控制发送失败: %v", ifName, err)
	}

	return err
}

// 执行波浪动画 - 支持设备类型
func startWaveAnimation(ifName string, speed int, handType string, handId uint32) {
	if speed <= 0 {
		speed = 500 // 默认速度
	}

	// 如果未指定接口，使用默认接口
	if ifName == "" {
		ifName = config.DefaultInterface
	}

	// 验证接口
	if !isValidInterface(ifName) {
		log.Printf("❌ 无法启动波浪动画: 无效的接口 %s", ifName)
		return
	}

	animationMutex.Lock()

	// 如果已经有动画在运行，先停止它
	if animationActive[ifName] {
		select {
		case stopAnimationMap[ifName] <- struct{}{}:
			// 发送成功
		default:
			// 通道已满，无需发送
		}

		stopAnimationMap[ifName] = make(chan struct{}, 1)
	}

	animationActive[ifName] = true
	animationMutex.Unlock()

	currentStopChannel := stopAnimationMap[ifName]

	go func() {
		defer func() {
			animationMutex.Lock()
			animationActive[ifName] = false
			animationMutex.Unlock()
			log.Printf("👋 %s 波浪动画已完成", ifName)
		}()

		// O7_MODIFIED: 根据设备类型使用不同的动画参数
		deviceType := config.DeviceType
		if handConfig, exists := handConfigs[ifName]; exists {
			deviceType = handConfig.DeviceType
		}

		var fingerOrder []int
		var open, close byte
		var fingerCount int

		if deviceType == DEVICE_TYPE_O7 {
			// O7型号: 7个关节
			fingerOrder = []int{0, 1, 2, 3, 4, 5, 6}
			fingerCount = 7
		} else {
			// L10型号: 6个关节
			fingerOrder = []int{0, 1, 2, 3, 4, 5}
			fingerCount = 6
		}

		open = byte(64)   // 0x40
		close = byte(192) // 0xC0

		log.Printf("🚀 开始 %s 波浪动画 (%s型号, %d个关节)",
			ifName, deviceType, fingerCount)

		// 动画循环
		for {
			select {
			case <-currentStopChannel:
				log.Printf("🛑 %s 波浪动画被用户停止", ifName)
				return
			default:
				// 波浪张开
				for _, idx := range fingerOrder {
					pose := make([]byte, fingerCount)
					for j := 0; j < fingerCount; j++ {
						if j == idx {
							pose[j] = open
						} else {
							pose[j] = close
						}
					}

					if err := sendFingerPose(ifName, pose, handType, handId); err != nil {
						log.Printf("%s 动画发送失败: %v", ifName, err)
						return
					}

					delay := time.Duration(speed) * time.Millisecond

					select {
					case <-currentStopChannel:
						log.Printf("🛑 %s 波浪动画被用户停止", ifName)
						return
					case <-time.After(delay):
						// 继续执行
					}
				}

				// 波浪握拳
				for _, idx := range fingerOrder {
					pose := make([]byte, fingerCount)
					for j := 0; j < fingerCount; j++ {
						if j == idx {
							pose[j] = close
						} else {
							pose[j] = open
						}
					}

					if err := sendFingerPose(ifName, pose, handType, handId); err != nil {
						log.Printf("%s 动画发送失败: %v", ifName, err)
						return
					}

					delay := time.Duration(speed) * time.Millisecond

					select {
					case <-currentStopChannel:
						log.Printf("🛑 %s 波浪动画被用户停止", ifName)
						return
					case <-time.After(delay):
						// 继续执行
					}
				}
			}
		}
	}()
}

// 执行横向摆动动画 - 支持设备类型
func startSwayAnimation(ifName string, speed int, handType string, handId uint32) {
	if speed <= 0 {
		speed = 500 // 默认速度
	}

	// 如果未指定接口，使用默认接口
	if ifName == "" {
		ifName = config.DefaultInterface
	}

	// 验证接口
	if !isValidInterface(ifName) {
		log.Printf("❌ 无法启动摆动动画: 无效的接口 %s", ifName)
		return
	}

	animationMutex.Lock()

	if animationActive[ifName] {
		select {
		case stopAnimationMap[ifName] <- struct{}{}:
			// 发送成功
		default:
			// 通道已满，无需发送
		}

		stopAnimationMap[ifName] = make(chan struct{}, 1)
	}

	animationActive[ifName] = true
	animationMutex.Unlock()

	currentStopChannel := stopAnimationMap[ifName]

	go func() {
		defer func() {
			animationMutex.Lock()
			animationActive[ifName] = false
			animationMutex.Unlock()
			log.Printf("🔄 %s 横向摆动动画已完成", ifName)
		}()

		// O7_MODIFIED: 根据设备类型执行不同的动画
		deviceType := config.DeviceType
		if handConfig, exists := handConfigs[ifName]; exists {
			deviceType = handConfig.DeviceType
		}

		// 默认姿势 - 对两种设备类型都适用
		leftPose := []byte{48, 48, 48, 48}      // 0x30
		rightPose := []byte{208, 208, 208, 208} // 0xD0

		log.Printf("🚀 开始 %s 横向摆动动画 (%s型号)", ifName, deviceType)

		// 动画循环
		for {
			select {
			case <-currentStopChannel:
				log.Printf("🛑 %s 横向摆动动画被用户停止", ifName)
				return
			default:
				// 向左移动
				if err := sendPalmPose(ifName, leftPose, handType, handId); err != nil {
					log.Printf("%s 动画发送失败: %v", ifName, err)
					return
				}

				delay := time.Duration(speed) * time.Millisecond

				select {
				case <-currentStopChannel:
					log.Printf("🛑 %s 横向摆动动画被用户停止", ifName)
					return
				case <-time.After(delay):
					// 继续执行
				}

				// 向右移动
				if err := sendPalmPose(ifName, rightPose, handType, handId); err != nil {
					log.Printf("%s 动画发送失败: %v", ifName, err)
					return
				}

				select {
				case <-currentStopChannel:
					log.Printf("🛑 %s 横向摆动动画被用户停止", ifName)
					return
				case <-time.After(delay):
					// 继续执行
				}
			}
		}
	}()
}

// 停止所有动画
func stopAllAnimations(ifName string) {
	// 如果未指定接口，停止所有接口的动画
	if ifName == "" {
		for _, validIface := range config.AvailableInterfaces {
			stopAllAnimations(validIface)
		}
		return
	}

	// 验证接口
	if !isValidInterface(ifName) {
		log.Printf("⚠️ 尝试停止无效接口的动画: %s", ifName)
		return
	}

	animationMutex.Lock()
	defer animationMutex.Unlock()

	if animationActive[ifName] {
		select {
		case stopAnimationMap[ifName] <- struct{}{}:
			log.Printf("✅ 已发送停止 %s 动画信号", ifName)
		default:
			stopAnimationMap[ifName] = make(chan struct{}, 1)
			stopAnimationMap[ifName] <- struct{}{}
			log.Printf("⚠️ %s 通道重置后发送了停止信号", ifName)
		}

		animationActive[ifName] = false

		go func() {
			time.Sleep(100 * time.Millisecond)
			resetToDefaultPose(ifName)
		}()
	} else {
		log.Printf("ℹ️ %s 当前没有运行中的动画", ifName)
	}
}

// 重置到默认姿势
func resetToDefaultPose(ifName string) {
	// 如果未指定接口，重置所有接口
	if ifName == "" {
		for _, validIface := range config.AvailableInterfaces {
			resetToDefaultPose(validIface)
		}
		return
	}

	// 验证接口
	if !isValidInterface(ifName) {
		log.Printf("⚠️ 尝试重置无效接口: %s", ifName)
		return
	}

	defaultFingerPose := []byte{64, 64, 64, 64, 64, 64}
	defaultPalmPose := []byte{128, 128, 128, 128}

	// 获取当前接口的手型配置
	handConfig := getHandConfig(ifName)

	if err := sendFingerPose(ifName, defaultFingerPose, handConfig.HandType, handConfig.HandId); err != nil {
		log.Printf("%s 重置手指姿势失败: %v", ifName, err)
	}

	if err := sendPalmPose(ifName, defaultPalmPose, handConfig.HandType, handConfig.HandId); err != nil {
		log.Printf("%s 重置掌部姿势失败: %v", ifName, err)
	}

	log.Printf("✅ 已重置 %s 到默认姿势", ifName)
}

// 读取传感器数据 (模拟)
func readSensorData() {
	go func() {
		for {
			sensorMutex.Lock()
			// 为每个接口模拟压力数据 (0-100)
			for _, ifName := range config.AvailableInterfaces {
				if sensorData, exists := sensorDataMap[ifName]; exists {
					sensorData.Thumb = rand.Intn(101)
					sensorData.Index = rand.Intn(101)
					sensorData.Middle = rand.Intn(101)
					sensorData.Ring = rand.Intn(101)
					sensorData.Pinky = rand.Intn(101)
					sensorData.LastUpdate = time.Now()
				}
			}
			sensorMutex.Unlock()

			time.Sleep(500 * time.Millisecond)
		}
	}()
}

// 检查 CAN 服务状态
func checkCanServiceStatus() map[string]bool {
	resp, err := http.Get(config.CanServiceURL + "/api/status")
	if err != nil {
		log.Printf("❌ CAN 服务状态检查失败: %v", err)
		result := make(map[string]bool)
		for _, ifName := range config.AvailableInterfaces {
			result[ifName] = false
		}
		return result
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("❌ CAN 服务返回非正常状态: %d", resp.StatusCode)
		result := make(map[string]bool)
		for _, ifName := range config.AvailableInterfaces {
			result[ifName] = false
		}
		return result
	}

	var statusResp ApiResponse
	if err := json.NewDecoder(resp.Body).Decode(&statusResp); err != nil {
		log.Printf("❌ 解析 CAN 服务状态失败: %v", err)
		result := make(map[string]bool)
		for _, ifName := range config.AvailableInterfaces {
			result[ifName] = false
		}
		return result
	}

	// 检查状态数据
	result := make(map[string]bool)
	for _, ifName := range config.AvailableInterfaces {
		result[ifName] = false
	}

	// 从响应中获取各接口状态
	if statusData, ok := statusResp.Data.(map[string]interface{}); ok {
		if interfaces, ok := statusData["interfaces"].(map[string]interface{}); ok {
			for ifName, ifStatus := range interfaces {
				if status, ok := ifStatus.(map[string]interface{}); ok {
					if active, ok := status["active"].(bool); ok {
						result[ifName] = active
					}
				}
			}
		}
	}

	return result
}

// API 路由设置
func setupRoutes(r *gin.Engine) {
	r.StaticFile("/", "./static/index.html")
	r.Static("/static", "./static")

	api := r.Group("/api")
	{
		// O7_MODIFIED: 添加设备类型检查端点
		api.GET("/device-type", func(c *gin.Context) {
			c.JSON(http.StatusOK, ApiResponse{
				Status: "success",
				Data: map[string]interface{}{
					"deviceType": config.DeviceType,
				},
			})
		})

		// 手型设置 API
		api.POST("/hand-type", func(c *gin.Context) {
			var req HandTypeRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, ApiResponse{
					Status: "error",
					Error:  "无效的手型设置请求: " + err.Error(),
				})
				return
			}

			// 验证接口
			if !isValidInterface(req.Interface) {
				c.JSON(http.StatusBadRequest, ApiResponse{
					Status: "error",
					Error:  fmt.Sprintf("无效的接口 %s，可用接口: %v", req.Interface, config.AvailableInterfaces),
				})
				return
			}

			// 验证手型ID
			if req.HandType == "left" && req.HandId != HAND_TYPE_LEFT {
				req.HandId = HAND_TYPE_LEFT
			} else if req.HandType == "right" && req.HandId != HAND_TYPE_RIGHT {
				req.HandId = HAND_TYPE_RIGHT
			}

			// 设置手型配置
			setHandConfig(req.Interface, req.HandType, req.HandId)

			handTypeName := "右手"
			if req.HandType == "left" {
				handTypeName = "左手"
			}

			c.JSON(http.StatusOK, ApiResponse{
				Status:  "success",
				Message: fmt.Sprintf("接口 %s 手型已设置为%s (0x%X)", req.Interface, handTypeName, req.HandId),
				Data: map[string]interface{}{
					"interface":  req.Interface,
					"handType":   req.HandType,
					"handId":     req.HandId,
					"deviceType": handConfigs[req.Interface].DeviceType,
				},
			})
		})

		// 手指姿态 API - 更新支持设备类型
		api.POST("/fingers", func(c *gin.Context) {
			var req FingerPoseRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, ApiResponse{
					Status: "error",
					Error:  "无效的手指姿态数据: " + err.Error(),
				})
				return
			}

			// 验证每个值是否在范围内
			for _, v := range req.Pose {
				if v < 0 || v > 255 {
					c.JSON(http.StatusBadRequest, ApiResponse{
						Status: "error",
						Error:  "手指姿态值必须在 0-255 范围内",
					})
					return
				}
			}

			// 如果未指定接口，使用默认接口
			if req.Interface == "" {
				req.Interface = config.DefaultInterface
			}

			// 验证接口
			if !isValidInterface(req.Interface) {
				c.JSON(http.StatusBadRequest, ApiResponse{
					Status: "error",
					Error:  fmt.Sprintf("无效的接口 %s，可用接口: %v", req.Interface, config.AvailableInterfaces),
				})
				return
			}

			// 如果指定了设备类型，更新接口的设备类型
			if req.DeviceType != "" {
				if req.DeviceType != DEVICE_TYPE_L10 && req.DeviceType != DEVICE_TYPE_O7 {
					c.JSON(http.StatusBadRequest, ApiResponse{
						Status: "error",
						Error:  fmt.Sprintf("无效的设备类型 %s，有效类型: [L10, O7]", req.DeviceType),
					})
					return
				}

				// 更新设备类型
				handConfigMutex.Lock()
				if handConfig, exists := handConfigs[req.Interface]; exists {
					handConfig.DeviceType = req.DeviceType
				}
				handConfigMutex.Unlock()
			}

			stopAllAnimations(req.Interface)

			if err := sendFingerPose(req.Interface, req.Pose, req.HandType, req.HandId); err != nil {
				c.JSON(http.StatusInternalServerError, ApiResponse{
					Status: "error",
					Error:  "发送手指姿态失败: " + err.Error(),
				})
				return
			}

			// 获取当前设备类型
			deviceType := config.DeviceType
			if handConfig, exists := handConfigs[req.Interface]; exists {
				deviceType = handConfig.DeviceType
			}

			c.JSON(http.StatusOK, ApiResponse{
				Status:  "success",
				Message: "手指姿态指令发送成功",
				Data: map[string]interface{}{
					"interface":  req.Interface,
					"pose":       req.Pose,
					"deviceType": deviceType,
				},
			})
		})

		// 掌部姿态 API - 更新支持设备类型
		api.POST("/palm", func(c *gin.Context) {
			var req PalmPoseRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, ApiResponse{
					Status: "error",
					Error:  "无效的掌部姿态数据: " + err.Error(),
				})
				return
			}

			// 验证每个值是否在范围内
			for _, v := range req.Pose {
				if v < 0 || v > 255 {
					c.JSON(http.StatusBadRequest, ApiResponse{
						Status: "error",
						Error:  "掌部姿态值必须在 0-255 范围内",
					})
					return
				}
			}

			// 如果未指定接口，使用默认接口
			if req.Interface == "" {
				req.Interface = config.DefaultInterface
			}

			// 验证接口
			if !isValidInterface(req.Interface) {
				c.JSON(http.StatusBadRequest, ApiResponse{
					Status: "error",
					Error:  fmt.Sprintf("无效的接口 %s，可用接口: %v", req.Interface, config.AvailableInterfaces),
				})
				return
			}

			// 如果指定了设备类型，更新接口的设备类型
			if req.DeviceType != "" {
				if req.DeviceType != DEVICE_TYPE_L10 && req.DeviceType != DEVICE_TYPE_O7 {
					c.JSON(http.StatusBadRequest, ApiResponse{
						Status: "error",
						Error:  fmt.Sprintf("无效的设备类型 %s，有效类型: [L10, O7]", req.DeviceType),
					})
					return
				}

				// 更新设备类型
				handConfigMutex.Lock()
				if handConfig, exists := handConfigs[req.Interface]; exists {
					handConfig.DeviceType = req.DeviceType
				}
				handConfigMutex.Unlock()
			}

			stopAllAnimations(req.Interface)

			if err := sendPalmPose(req.Interface, req.Pose, req.HandType, req.HandId); err != nil {
				c.JSON(http.StatusInternalServerError, ApiResponse{
					Status: "error",
					Error:  "发送掌部姿态失败: " + err.Error(),
				})
				return
			}

			// 获取当前设备类型
			deviceType := config.DeviceType
			if handConfig, exists := handConfigs[req.Interface]; exists {
				deviceType = handConfig.DeviceType
			}

			c.JSON(http.StatusOK, ApiResponse{
				Status:  "success",
				Message: "掌部姿态指令发送成功",
				Data: map[string]interface{}{
					"interface":  req.Interface,
					"pose":       req.Pose,
					"deviceType": deviceType,
				},
			})
		})

		// O7_MODIFIED: 添加关节速度控制API端点
		api.POST("/speeds", func(c *gin.Context) {
			var req SpeedRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, ApiResponse{
					Status: "error",
					Error:  "无效的速度数据: " + err.Error(),
				})
				return
			}

			// 验证每个值是否在范围内
			for _, v := range req.Speeds {
				if v < 0 || v > 255 {
					c.JSON(http.StatusBadRequest, ApiResponse{
						Status: "error",
						Error:  "速度值必须在 0-255 范围内",
					})
					return
				}
			}

			// 如果未指定接口，使用默认接口
			if req.Interface == "" {
				req.Interface = config.DefaultInterface
			}

			// 验证接口
			if !isValidInterface(req.Interface) {
				c.JSON(http.StatusBadRequest, ApiResponse{
					Status: "error",
					Error:  fmt.Sprintf("无效的接口 %s，可用接口: %v", req.Interface, config.AvailableInterfaces),
				})
				return
			}

			// 确定设备类型
			deviceType := config.DeviceType
			if req.DeviceType != "" {
				// 如果请求指定了设备类型
				if req.DeviceType != DEVICE_TYPE_L10 && req.DeviceType != DEVICE_TYPE_O7 {
					c.JSON(http.StatusBadRequest, ApiResponse{
						Status: "error",
						Error:  fmt.Sprintf("无效的设备类型 %s，有效类型: [L10, O7]", req.DeviceType),
					})
					return
				}
				deviceType = req.DeviceType

				// 更新设备类型
				handConfigMutex.Lock()
				if handConfig, exists := handConfigs[req.Interface]; exists {
					handConfig.DeviceType = req.DeviceType
				}
				handConfigMutex.Unlock()
			} else if handConfig, exists := handConfigs[req.Interface]; exists {
				deviceType = handConfig.DeviceType
			}

			// 验证速度数据长度
			if deviceType == DEVICE_TYPE_O7 {
				if len(req.Speeds) != 7 && len(req.Speeds) != 6 {
					c.JSON(http.StatusBadRequest, ApiResponse{
						Status: "error",
						Error:  "O7设备需要7个关节速度值(或兼容L10的6个值)",
					})
					return
				}
			} else {
				if len(req.Speeds) != 5 {
					c.JSON(http.StatusBadRequest, ApiResponse{
						Status: "error",
						Error:  "L10设备需要5个手指的速度值",
					})
					return
				}
			}

			if err := sendJointSpeeds(req.Interface, req.Speeds, req.HandType, req.HandId); err != nil {
				c.JSON(http.StatusInternalServerError, ApiResponse{
					Status: "error",
					Error:  "发送关节速度失败: " + err.Error(),
				})
				return
			}

			c.JSON(http.StatusOK, ApiResponse{
				Status:  "success",
				Message: "关节速度指令发送成功",
				Data: map[string]interface{}{
					"interface":  req.Interface,
					"speeds":     req.Speeds,
					"deviceType": deviceType,
				},
			})
		})

		// 预设姿势 API - 更新支持手型
		api.POST("/preset/:pose", func(c *gin.Context) {
			pose := c.Param("pose")

			// 从查询参数获取接口名称和手型
			ifName := c.Query("interface")
			handType := c.Query("handType")

			if ifName == "" {
				ifName = config.DefaultInterface
			}

			// 验证接口
			if !isValidInterface(ifName) {
				c.JSON(http.StatusBadRequest, ApiResponse{
					Status: "error",
					Error:  fmt.Sprintf("无效的接口 %s，可用接口: %v", ifName, config.AvailableInterfaces),
				})
				return
			}

			stopAllAnimations(ifName)

			var fingerPose []byte
			var message string

			switch pose {
			case "fist":
				fingerPose = []byte{64, 64, 64, 64, 64, 64}
				message = "已设置握拳姿势"
			case "open":
				fingerPose = []byte{192, 192, 192, 192, 192, 192}
				message = "已设置完全张开姿势"
			case "pinch":
				fingerPose = []byte{120, 120, 64, 64, 64, 64}
				message = "已设置捏取姿势"
			case "thumbsup":
				fingerPose = []byte{64, 192, 192, 192, 192, 64}
				message = "已设置竖起大拇指姿势"
			case "point":
				fingerPose = []byte{192, 64, 192, 192, 192, 64}
				message = "已设置食指指点姿势"
			// 数字手势
			case "1":
				fingerPose = []byte{192, 64, 192, 192, 192, 64}
				message = "已设置数字1手势"
			case "2":
				fingerPose = []byte{192, 64, 64, 192, 192, 64}
				message = "已设置数字2手势"
			case "3":
				fingerPose = []byte{192, 64, 64, 64, 192, 64}
				message = "已设置数字3手势"
			case "4":
				fingerPose = []byte{192, 64, 64, 64, 64, 64}
				message = "已设置数字4手势"
			case "5":
				fingerPose = []byte{192, 192, 192, 192, 192, 192}
				message = "已设置数字5手势"
			case "6":
				fingerPose = []byte{64, 192, 192, 192, 192, 64}
				message = "已设置数字6手势"
			case "7":
				fingerPose = []byte{64, 64, 192, 192, 192, 64}
				message = "已设置数字7手势"
			case "8":
				fingerPose = []byte{64, 64, 64, 192, 192, 64}
				message = "已设置数字8手势"
			case "9":
				fingerPose = []byte{64, 64, 64, 64, 192, 64}
				message = "已设置数字9手势"
			default:
				c.JSON(http.StatusBadRequest, ApiResponse{
					Status: "error",
					Error:  "无效的预设姿势",
				})
				return
			}

			// 解析手型ID（从查询参数或使用接口配置）
			handId := uint32(0)
			if handType != "" {
				handId = parseHandType(handType, 0, ifName)
			}

			if err := sendFingerPose(ifName, fingerPose, handType, handId); err != nil {
				c.JSON(http.StatusInternalServerError, ApiResponse{
					Status: "error",
					Error:  "设置预设姿势失败: " + err.Error(),
				})
				return
			}

			c.JSON(http.StatusOK, ApiResponse{
				Status:  "success",
				Message: message,
				Data:    map[string]interface{}{"interface": ifName, "pose": fingerPose},
			})
		})

		// 动画控制 API - 更新支持手型
		api.POST("/animation", func(c *gin.Context) {
			var req AnimationRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, ApiResponse{
					Status: "error",
					Error:  "无效的动画请求: " + err.Error(),
				})
				return
			}

			// 如果未指定接口，使用默认接口
			if req.Interface == "" {
				req.Interface = config.DefaultInterface
			}

			// 验证接口
			if !isValidInterface(req.Interface) {
				c.JSON(http.StatusBadRequest, ApiResponse{
					Status: "error",
					Error:  fmt.Sprintf("无效的接口 %s，可用接口: %v", req.Interface, config.AvailableInterfaces),
				})
				return
			}

			// 停止当前动画
			stopAllAnimations(req.Interface)

			// 如果是停止命令，直接返回
			if req.Type == "stop" {
				c.JSON(http.StatusOK, ApiResponse{
					Status:  "success",
					Message: fmt.Sprintf("%s 动画已停止", req.Interface),
				})
				return
			}

			// 处理速度参数
			if req.Speed <= 0 {
				req.Speed = 500 // 默认速度
			}

			// 根据类型启动动画
			switch req.Type {
			case "wave":
				startWaveAnimation(req.Interface, req.Speed, req.HandType, req.HandId)
				c.JSON(http.StatusOK, ApiResponse{
					Status:  "success",
					Message: fmt.Sprintf("%s 波浪动画已启动", req.Interface),
					Data:    map[string]interface{}{"interface": req.Interface, "speed": req.Speed},
				})
			case "sway":
				startSwayAnimation(req.Interface, req.Speed, req.HandType, req.HandId)
				c.JSON(http.StatusOK, ApiResponse{
					Status:  "success",
					Message: fmt.Sprintf("%s 横向摆动动画已启动", req.Interface),
					Data:    map[string]interface{}{"interface": req.Interface, "speed": req.Speed},
				})
			default:
				c.JSON(http.StatusBadRequest, ApiResponse{
					Status: "error",
					Error:  "无效的动画类型",
				})
			}
		})

		// 获取传感器数据 API
		api.GET("/sensors", func(c *gin.Context) {
			// 从查询参数获取接口名称
			ifName := c.Query("interface")

			sensorMutex.RLock()
			defer sensorMutex.RUnlock()

			if ifName != "" {
				// 验证接口
				if !isValidInterface(ifName) {
					c.JSON(http.StatusBadRequest, ApiResponse{
						Status: "error",
						Error:  fmt.Sprintf("无效的接口 %s，可用接口: %v", ifName, config.AvailableInterfaces),
					})
					return
				}

				// 请求特定接口的数据
				if sensorData, ok := sensorDataMap[ifName]; ok {
					c.JSON(http.StatusOK, ApiResponse{
						Status: "success",
						Data:   sensorData,
					})
				} else {
					c.JSON(http.StatusInternalServerError, ApiResponse{
						Status: "error",
						Error:  "传感器数据不存在",
					})
				}
			} else {
				// 返回所有接口的数据
				c.JSON(http.StatusOK, ApiResponse{
					Status: "success",
					Data:   sensorDataMap,
				})
			}
		})

		// 系统状态 API - 更新包含手型配置
		api.GET("/status", func(c *gin.Context) {
			animationMutex.Lock()
			animationStatus := make(map[string]bool)
			for _, ifName := range config.AvailableInterfaces {
				animationStatus[ifName] = animationActive[ifName]
			}
			animationMutex.Unlock()

			// 检查 CAN 服务状态
			canStatus := checkCanServiceStatus()

			// 获取手型配置
			handConfigMutex.RLock()
			handConfigsData := make(map[string]interface{})
			for ifName, handConfig := range handConfigs {
				handConfigsData[ifName] = map[string]interface{}{
					"handType": handConfig.HandType,
					"handId":   handConfig.HandId,
				}
			}
			handConfigMutex.RUnlock()

			interfaceStatuses := make(map[string]interface{})
			for _, ifName := range config.AvailableInterfaces {
				interfaceStatuses[ifName] = map[string]interface{}{
					"active":          canStatus[ifName],
					"animationActive": animationStatus[ifName],
					"handConfig":      handConfigsData[ifName],
				}
			}

			c.JSON(http.StatusOK, ApiResponse{
				Status: "success",
				Data: map[string]interface{}{
					"interfaces":          interfaceStatuses,
					"uptime":              time.Since(serverStartTime).String(),
					"canServiceURL":       config.CanServiceURL,
					"defaultInterface":    config.DefaultInterface,
					"availableInterfaces": config.AvailableInterfaces,
					"activeInterfaces":    len(canStatus),
					"handConfigs":         handConfigsData,
				},
			})
		})

		// 获取可用接口列表 API - 修复数据格式
		api.GET("/interfaces", func(c *gin.Context) {
			// 确保返回前端期望的数据格式
			responseData := map[string]interface{}{
				"availableInterfaces": config.AvailableInterfaces,
				"defaultInterface":    config.DefaultInterface,
			}

			c.JSON(http.StatusOK, ApiResponse{
				Status: "success",
				Data:   responseData,
			})
		})

		// 获取手型配置 API - 新增
		api.GET("/hand-configs", func(c *gin.Context) {
			handConfigMutex.RLock()
			defer handConfigMutex.RUnlock()

			result := make(map[string]interface{})
			for _, ifName := range config.AvailableInterfaces {
				if handConfig, exists := handConfigs[ifName]; exists {
					result[ifName] = map[string]interface{}{
						"handType": handConfig.HandType,
						"handId":   handConfig.HandId,
					}
				} else {
					// 返回默认配置
					result[ifName] = map[string]interface{}{
						"handType": "right",
						"handId":   HAND_TYPE_RIGHT,
					}
				}
			}

			c.JSON(http.StatusOK, ApiResponse{
				Status: "success",
				Data:   result,
			})
		})

		// 健康检查端点 - 新增，用于调试
		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, ApiResponse{
				Status:  "success",
				Message: "CAN Control Service is running",
				Data: map[string]interface{}{
					"timestamp":           time.Now(),
					"availableInterfaces": config.AvailableInterfaces,
					"defaultInterface":    config.DefaultInterface,
					"serviceVersion":      "1.0.0-hand-type-support",
				},
			})
		})
	}
}

func printUsage() {
	fmt.Println("CAN Control Service with Hand Type Support")
	fmt.Println("Usage:")
	fmt.Println("  -can-url string         CAN 服务的 URL (default: http://10.211.55.7:8080)")
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
	fmt.Println("  CAN_SERVICE_URL=http://localhost:8080 ./control-service")
}

func main() {
	// 检查是否请求帮助
	if len(os.Args) > 1 && (os.Args[1] == "-h" || os.Args[1] == "--help") {
		printUsage()
		return
	}

	// 解析配置
	config = parseConfig()

	// 验证配置
	if len(config.AvailableInterfaces) == 0 {
		log.Fatal("❌ 没有可用的 CAN 接口")
	}

	if config.DefaultInterface == "" {
		log.Fatal("❌ 没有设置默认 CAN 接口")
	}

	// 记录启动时间
	serverStartTime = time.Now()

	log.Printf("🚀 启动 CAN 控制服务 (支持左右手配置)")

	// 初始化随机数种子
	rand.Seed(time.Now().UnixNano())

	// 初始化服务
	initService()

	// 启动传感器数据模拟
	readSensorData()

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

	// 设置 API 路由
	setupRoutes(r)

	// 启动服务器
	log.Printf("🌐 CAN 控制服务运行在 http://localhost:%s", config.WebPort)
	log.Printf("📡 连接到 CAN 服务: %s", config.CanServiceURL)
	log.Printf("🎯 默认接口: %s", config.DefaultInterface)
	log.Printf("🔌 可用接口: %v", config.AvailableInterfaces)
	log.Printf("🤖 支持左右手动态配置")

	if err := r.Run(":" + config.WebPort); err != nil {
		log.Fatalf("❌ 服务启动失败: %v", err)
	}
}

// 在 base 基础上进行 ±delta 的扰动，范围限制在 [0, 255]
func perturb(base byte, delta int) byte {
	offset := rand.Intn(2*delta+1) - delta
	v := int(base) + offset
	if v < 0 {
		v = 0
	}
	if v > 255 {
		v = 255
	}
	return byte(v)
}
