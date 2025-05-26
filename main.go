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

const HAND_TYPE_LEFT = 0x28
const HAND_TYPE_RIGHT = 0x27

// API 请求结构体 - 添加手型支持
type FingerPoseRequest struct {
	Interface string `json:"interface,omitempty"`
	Pose      []byte `json:"pose" binding:"required,len=6"`
	HandType  string `json:"handType,omitempty"` // 新增: 手型类型
	HandId    uint32 `json:"handId,omitempty"`   // 新增: CAN ID
}

type PalmPoseRequest struct {
	Interface string `json:"interface,omitempty"`
	Pose      []byte `json:"pose" binding:"required,len=4"`
	HandType  string `json:"handType,omitempty"` // 新增: 手型类型
	HandId    uint32 `json:"handId,omitempty"`   // 新增: CAN ID
}

type AnimationRequest struct {
	Interface string `json:"interface,omitempty"`
	Type      string `json:"type" binding:"required,oneof=wave sway stop"`
	Speed     int    `json:"speed" binding:"min=0,max=2000"`
	HandType  string `json:"handType,omitempty"` // 新增: 手型类型
	HandId    uint32 `json:"handId,omitempty"`   // 新增: CAN ID
}

// 新增: 手型设置请求
type HandTypeRequest struct {
	Interface string `json:"interface" binding:"required"`
	HandType  string `json:"handType" binding:"required,oneof=left right"`
	HandId    uint32 `json:"handId" binding:"required"`
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
}

// 手型配置结构体
type HandConfig struct {
	HandType string `json:"handType"`
	HandId   uint32 `json:"handId"`
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
	flag.StringVar(&cfg.CanServiceURL, "can-url", "http://127.0.0.1:5260", "CAN 服务的 URL")
	flag.StringVar(&cfg.WebPort, "port", "9099", "Web 服务的端口")
	flag.StringVar(&cfg.DefaultInterface, "interface", "", "默认 CAN 接口")
	flag.StringVar(&canInterfacesFlag, "can-interfaces", "", "支持的 CAN 接口列表，用逗号分隔 (例如: can0,can1,vcan0)")
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

	// 创建默认配置（右手）
	handConfigs[ifName] = &HandConfig{
		HandType: "right",
		HandId:   HAND_TYPE_RIGHT,
	}

	log.Printf("🆕 为接口 %s 创建默认手型配置: 右手 (0x%X)", ifName, HAND_TYPE_RIGHT)
	return handConfigs[ifName]
}

// 设置手型配置
func setHandConfig(ifName, handType string, handId uint32) {
	handConfigMutex.Lock()
	defer handConfigMutex.Unlock()

	handConfigs[ifName] = &HandConfig{
		HandType: handType,
		HandId:   handId,
	}

	log.Printf("🔧 接口 %s 手型配置已更新: %s (0x%X)", ifName, handType, handId)
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

// 发送手指姿态指令 - 支持手型参数
func sendFingerPose(ifName string, pose []byte, handType string, handId uint32) error {
	if len(pose) != 6 {
		return fmt.Errorf("无效的姿态数据长度，需要 6 个字节")
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

	// 构造 CAN 消息
	msg := CanMessage{
		Interface: ifName,
		ID:        canId, // 使用动态的手型ID
		Data:      append([]byte{0x01}, perturbedPose...),
	}

	err := sendToCanService(msg)
	if err == nil {
		handTypeName := "右手"
		if canId == HAND_TYPE_LEFT {
			handTypeName = "左手"
		}
		log.Printf("✅ %s (%s, 0x%X) 手指动作已发送: [%X %X %X %X %X %X]",
			ifName, handTypeName, canId, perturbedPose[0], perturbedPose[1], perturbedPose[2],
			perturbedPose[3], perturbedPose[4], perturbedPose[5])
	} else {
		log.Printf("❌ %s 手指控制发送失败: %v", ifName, err)
	}

	return err
}

// 发送掌部姿态指令 - 支持手型参数
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

	// 构造 CAN 消息
	msg := CanMessage{
		Interface: ifName,
		ID:        canId, // 使用动态的手型ID
		Data:      append([]byte{0x04}, perturbedPose...),
	}

	err := sendToCanService(msg)
	if err == nil {
		handTypeName := "右手"
		if canId == HAND_TYPE_LEFT {
			handTypeName = "左手"
		}
		log.Printf("✅ %s (%s, 0x%X) 掌部姿态已发送: [%X %X %X %X]",
			ifName, handTypeName, canId, perturbedPose[0], perturbedPose[1], perturbedPose[2], perturbedPose[3])

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

// 执行波浪动画 - 支持手型参数
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

		fingerOrder := []int{0, 1, 2, 3, 4, 5}
		open := byte(64)   // 0x40
		close := byte(192) // 0xC0

		log.Printf("🚀 开始 %s 波浪动画", ifName)

		// 动画循环
		for {
			select {
			case <-currentStopChannel:
				log.Printf("🛑 %s 波浪动画被用户停止", ifName)
				return
			default:
				// 波浪张开
				for _, idx := range fingerOrder {
					pose := make([]byte, 6)
					for j := 0; j < 6; j++ {
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
					pose := make([]byte, 6)
					for j := 0; j < 6; j++ {
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

// 执行横向摆动动画 - 支持手型参数
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

		leftPose := []byte{48, 48, 48, 48}      // 0x30
		rightPose := []byte{208, 208, 208, 208} // 0xD0

		log.Printf("🚀 开始 %s 横向摆动动画", ifName)

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
		// 手型设置 API - 新增
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
					"interface": req.Interface,
					"handType":  req.HandType,
					"handId":    req.HandId,
				},
			})
		})

		// 手指姿态 API - 更新支持手型
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

			stopAllAnimations(req.Interface)

			if err := sendFingerPose(req.Interface, req.Pose, req.HandType, req.HandId); err != nil {
				c.JSON(http.StatusInternalServerError, ApiResponse{
					Status: "error",
					Error:  "发送手指姿态失败: " + err.Error(),
				})
				return
			}

			c.JSON(http.StatusOK, ApiResponse{
				Status:  "success",
				Message: "手指姿态指令发送成功",
				Data:    map[string]interface{}{"interface": req.Interface, "pose": req.Pose},
			})
		})

		// 掌部姿态 API - 更新支持手型
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

			stopAllAnimations(req.Interface)

			if err := sendPalmPose(req.Interface, req.Pose, req.HandType, req.HandId); err != nil {
				c.JSON(http.StatusInternalServerError, ApiResponse{
					Status: "error",
					Error:  "发送掌部姿态失败: " + err.Error(),
				})
				return
			}

			c.JSON(http.StatusOK, ApiResponse{
				Status:  "success",
				Message: "掌部姿态指令发送成功",
				Data:    map[string]interface{}{"interface": req.Interface, "pose": req.Pose},
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
