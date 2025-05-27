package hands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"hands/config"
	"hands/define"
	"log"
	"math/rand/v2"
	"net/http"
	"strings"
	"sync"
	"time"
)

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

// 手型配置结构体
type HandConfig struct {
	HandType string `json:"handType"`
	HandId   uint32 `json:"handId"`
}

var (
	HandConfigMutex sync.RWMutex
	HandConfigs     map[string]*HandConfig // 每个接口的手型配置

	SensorDataMap    map[string]*SensorData // 每个接口的传感器数据
	SensorMutex      sync.RWMutex
	AnimationActive  map[string]bool // 每个接口的动画状态
	AnimationMutex   sync.Mutex
	StopAnimationMap map[string]chan struct{} // 每个接口的停止动画通道
)

func InitHands() {
	// 初始化传感器数据映射
	SensorDataMap = make(map[string]*SensorData)
	for _, ifName := range config.Config.AvailableInterfaces {
		SensorDataMap[ifName] = &SensorData{
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
	AnimationActive = make(map[string]bool)
	StopAnimationMap = make(map[string]chan struct{})
	for _, ifName := range config.Config.AvailableInterfaces {
		AnimationActive[ifName] = false
		StopAnimationMap[ifName] = make(chan struct{}, 1)
	}

	HandConfigs = make(map[string]*HandConfig)
}

func SetHandConfig(ifName, handType string, handId uint32) {
	HandConfigMutex.Lock()
	defer HandConfigMutex.Unlock()

	HandConfigs[ifName] = &HandConfig{
		HandType: handType,
		HandId:   handId,
	}

	log.Printf("🔧 接口 %s 手型配置已更新: %s (0x%X)", ifName, handType, handId)
}

func GetHandConfig(ifName string) *HandConfig {
	HandConfigMutex.RLock()
	if handConfig, exists := HandConfigs[ifName]; exists {
		HandConfigMutex.RUnlock()
		return handConfig
	}
	HandConfigMutex.RUnlock()

	// 创建默认配置
	HandConfigMutex.Lock()
	defer HandConfigMutex.Unlock()

	// 再次检查（双重检查锁定）
	if handConfig, exists := HandConfigs[ifName]; exists {
		return handConfig
	}

	// 创建默认配置（右手）
	HandConfigs[ifName] = &HandConfig{
		HandType: "right",
		HandId:   define.HAND_TYPE_RIGHT,
	}

	log.Printf("🆕 为接口 %s 创建默认手型配置: 右手 (0x%X)", ifName, define.HAND_TYPE_RIGHT)
	return HandConfigs[ifName]
}

// 解析手型参数
func ParseHandType(handType string, handId uint32, ifName string) uint32 {
	// 如果提供了有效的 handId，直接使用
	if handId != 0 {
		return handId
	}

	// 根据 handType 字符串确定 ID
	switch strings.ToLower(handType) {
	case "left":
		return define.HAND_TYPE_LEFT
	case "right":
		return define.HAND_TYPE_RIGHT
	default:
		// 使用接口的配置
		handConfig := GetHandConfig(ifName)
		return handConfig.HandId
	}
}

// 验证接口是否可用
func IsValidInterface(ifName string) bool {
	for _, validIface := range config.Config.AvailableInterfaces {
		if ifName == validIface {
			return true
		}
	}
	return false
}

type CanMessage struct {
	Interface string `json:"interface"`
	ID        uint32 `json:"id"`
	Data      []byte `json:"data"`
}

// 发送手指姿态指令 - 支持手型参数
func SendFingerPose(ifName string, pose []byte, handType string, handId uint32) error {
	if len(pose) != 6 {
		return fmt.Errorf("无效的姿态数据长度，需要 6 个字节")
	}

	// 如果未指定接口，使用默认接口
	if ifName == "" {
		ifName = config.Config.DefaultInterface
	}

	// 验证接口
	if !IsValidInterface(ifName) {
		return fmt.Errorf("无效的接口 %s，可用接口: %v", ifName, config.Config.AvailableInterfaces)
	}

	// 解析手型 ID
	canId := ParseHandType(handType, handId, ifName)

	// 添加随机扰动
	perturbedPose := make([]byte, len(pose))
	for i, v := range pose {
		perturbedPose[i] = perturb(v, 5)
	}

	// 构造 CAN 消息
	msg := CanMessage{
		Interface: ifName,
		ID:        canId, // 使用动态的手型 ID
		Data:      append([]byte{0x01}, perturbedPose...),
	}

	err := sendToCanService(msg)
	if err == nil {
		handTypeName := "右手"
		if canId == define.HAND_TYPE_LEFT {
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

// 在 base 基础上进行 ±delta 的扰动，范围限制在 [0, 255]
func perturb(base byte, delta int) byte {
	offset := rand.IntN(2*delta+1) - delta
	v := int(base) + offset
	if v < 0 {
		v = 0
	}
	if v > 255 {
		v = 255
	}
	return byte(v)
}

// 发送请求到 CAN 服务
func sendToCanService(msg CanMessage) error {
	jsonData, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("JSON 编码错误: %v", err)
	}

	resp, err := http.Post(config.Config.CanServiceURL+"/api/can", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("CAN 服务请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp define.ApiResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			return fmt.Errorf("CAN 服务返回错误：HTTP %d", resp.StatusCode)
		}
		return fmt.Errorf("CAN 服务返回错误: %s", errResp.Error)
	}

	return nil
}

// 发送掌部姿态指令 - 支持手型参数
func SendPalmPose(ifName string, pose []byte, handType string, handId uint32) error {
	if len(pose) != 4 {
		return fmt.Errorf("无效的姿态数据长度，需要 4 个字节")
	}

	// 如果未指定接口，使用默认接口
	if ifName == "" {
		ifName = config.Config.DefaultInterface
	}

	// 验证接口
	if !IsValidInterface(ifName) {
		return fmt.Errorf("无效的接口 %s，可用接口: %v", ifName, config.Config.AvailableInterfaces)
	}

	// 解析手型 ID
	canId := ParseHandType(handType, handId, ifName)

	// 添加随机扰动
	perturbedPose := make([]byte, len(pose))
	for i, v := range pose {
		perturbedPose[i] = perturb(v, 8)
	}

	// 构造 CAN 消息
	msg := CanMessage{
		Interface: ifName,
		ID:        canId, // 使用动态的手型 ID
		Data:      append([]byte{0x04}, perturbedPose...),
	}

	err := sendToCanService(msg)
	if err == nil {
		handTypeName := "右手"
		if canId == define.HAND_TYPE_LEFT {
			handTypeName = "左手"
		}
		log.Printf("✅ %s (%s, 0x%X) 掌部姿态已发送: [%X %X %X %X]",
			ifName, handTypeName, canId, perturbedPose[0], perturbedPose[1], perturbedPose[2], perturbedPose[3])

		// 更新传感器数据中的掌部位置
		SensorMutex.Lock()
		if sensorData, exists := SensorDataMap[ifName]; exists {
			copy(sensorData.PalmPosition, perturbedPose)
			sensorData.LastUpdate = time.Now()
		}
		SensorMutex.Unlock()
	} else {
		log.Printf("❌ %s 掌部控制发送失败: %v", ifName, err)
	}

	return err
}

// 执行波浪动画 - 支持手型参数
func StartWaveAnimation(ifName string, speed int, handType string, handId uint32) {
	if speed <= 0 {
		speed = 500 // 默认速度
	}

	// 如果未指定接口，使用默认接口
	if ifName == "" {
		ifName = config.Config.DefaultInterface
	}

	// 验证接口
	if !IsValidInterface(ifName) {
		log.Printf("❌ 无法启动波浪动画: 无效的接口 %s", ifName)
		return
	}

	AnimationMutex.Lock()

	// 如果已经有动画在运行，先停止它
	if AnimationActive[ifName] {
		select {
		case StopAnimationMap[ifName] <- struct{}{}:
			// 发送成功
		default:
			// 通道已满，无需发送
		}

		StopAnimationMap[ifName] = make(chan struct{}, 1)
	}

	AnimationActive[ifName] = true
	AnimationMutex.Unlock()

	currentStopChannel := StopAnimationMap[ifName]

	go func() {
		defer func() {
			AnimationMutex.Lock()
			AnimationActive[ifName] = false
			AnimationMutex.Unlock()
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

					if err := SendFingerPose(ifName, pose, handType, handId); err != nil {
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

					if err := SendFingerPose(ifName, pose, handType, handId); err != nil {
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
func StartSwayAnimation(ifName string, speed int, handType string, handId uint32) {
	if speed <= 0 {
		speed = 500 // 默认速度
	}

	// 如果未指定接口，使用默认接口
	if ifName == "" {
		ifName = config.Config.DefaultInterface
	}

	// 验证接口
	if !IsValidInterface(ifName) {
		log.Printf("❌ 无法启动摆动动画: 无效的接口 %s", ifName)
		return
	}

	AnimationMutex.Lock()

	if AnimationActive[ifName] {
		select {
		case StopAnimationMap[ifName] <- struct{}{}:
			// 发送成功
		default:
			// 通道已满，无需发送
		}

		StopAnimationMap[ifName] = make(chan struct{}, 1)
	}

	AnimationActive[ifName] = true
	AnimationMutex.Unlock()

	currentStopChannel := StopAnimationMap[ifName]

	go func() {
		defer func() {
			AnimationMutex.Lock()
			AnimationActive[ifName] = false
			AnimationMutex.Unlock()
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
				if err := SendPalmPose(ifName, leftPose, handType, handId); err != nil {
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
				if err := SendPalmPose(ifName, rightPose, handType, handId); err != nil {
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
func StopAllAnimations(ifName string) {
	// 如果未指定接口，停止所有接口的动画
	if ifName == "" {
		for _, validIface := range config.Config.AvailableInterfaces {
			StopAllAnimations(validIface)
		}
		return
	}

	// 验证接口
	if !IsValidInterface(ifName) {
		log.Printf("⚠️ 尝试停止无效接口的动画: %s", ifName)
		return
	}

	AnimationMutex.Lock()
	defer AnimationMutex.Unlock()

	if AnimationActive[ifName] {
		select {
		case StopAnimationMap[ifName] <- struct{}{}:
			log.Printf("✅ 已发送停止 %s 动画信号", ifName)
		default:
			StopAnimationMap[ifName] = make(chan struct{}, 1)
			StopAnimationMap[ifName] <- struct{}{}
			log.Printf("⚠️ %s 通道重置后发送了停止信号", ifName)
		}

		AnimationActive[ifName] = false

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
		for _, validIface := range config.Config.AvailableInterfaces {
			resetToDefaultPose(validIface)
		}
		return
	}

	// 验证接口
	if !IsValidInterface(ifName) {
		log.Printf("⚠️ 尝试重置无效接口: %s", ifName)
		return
	}

	defaultFingerPose := []byte{64, 64, 64, 64, 64, 64}
	defaultPalmPose := []byte{128, 128, 128, 128}

	// 获取当前接口的手型配置
	handConfig := GetHandConfig(ifName)

	if err := SendFingerPose(ifName, defaultFingerPose, handConfig.HandType, handConfig.HandId); err != nil {
		log.Printf("%s 重置手指姿势失败: %v", ifName, err)
	}

	if err := SendPalmPose(ifName, defaultPalmPose, handConfig.HandType, handConfig.HandId); err != nil {
		log.Printf("%s 重置掌部姿势失败: %v", ifName, err)
	}

	log.Printf("✅ 已重置 %s 到默认姿势", ifName)
}

// 读取传感器数据 (模拟)
func ReadSensorData() {
	go func() {
		for {
			SensorMutex.Lock()
			// 为每个接口模拟压力数据 (0-100)
			for _, ifName := range config.Config.AvailableInterfaces {
				if sensorData, exists := SensorDataMap[ifName]; exists {
					sensorData.Thumb = rand.IntN(101)
					sensorData.Index = rand.IntN(101)
					sensorData.Middle = rand.IntN(101)
					sensorData.Ring = rand.IntN(101)
					sensorData.Pinky = rand.IntN(101)
					sensorData.LastUpdate = time.Now()
				}
			}
			SensorMutex.Unlock()

			time.Sleep(500 * time.Millisecond)
		}
	}()
}

// 检查 CAN 服务状态
func CheckCanServiceStatus() map[string]bool {
	resp, err := http.Get(config.Config.CanServiceURL + "/api/status")
	if err != nil {
		log.Printf("❌ CAN 服务状态检查失败: %v", err)
		result := make(map[string]bool)
		for _, ifName := range config.Config.AvailableInterfaces {
			result[ifName] = false
		}
		return result
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("❌ CAN 服务返回非正常状态：%d", resp.StatusCode)
		result := make(map[string]bool)
		for _, ifName := range config.Config.AvailableInterfaces {
			result[ifName] = false
		}
		return result
	}

	var statusResp define.ApiResponse
	if err := json.NewDecoder(resp.Body).Decode(&statusResp); err != nil {
		log.Printf("❌ 解析 CAN 服务状态失败: %v", err)
		result := make(map[string]bool)
		for _, ifName := range config.Config.AvailableInterfaces {
			result[ifName] = false
		}
		return result
	}

	// 检查状态数据
	result := make(map[string]bool)
	for _, ifName := range config.Config.AvailableInterfaces {
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
