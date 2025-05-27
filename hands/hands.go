package hands

import (
	"fmt"
	"hands/config"
	"hands/define"
	"log"
	"math/rand/v2"
	"slices"
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
)

func Init() {
	initSensorData()
	initAnimation()
	initHands()
}

func initHands() {
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
	return slices.Contains(config.Config.AvailableInterfaces, ifName)
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
