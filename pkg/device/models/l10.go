package models

import (
	"fmt"
	"log"
	"math/rand/v2"
	"sync"
	"time"

	"hands/define"
	"hands/pkg/communication"
	"hands/pkg/component"
	"hands/pkg/device"
)

// L10Hand L10 型号手部设备实现
type L10Hand struct {
	id              string
	model           string
	handType        define.HandType
	communicator    communication.Communicator
	components      map[device.ComponentType][]device.Component
	status          device.DeviceStatus
	mutex           sync.RWMutex
	canInterface    string                  // CAN 接口名称，如 "can0"
	animationEngine *device.AnimationEngine // 动画引擎
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

// NewL10Hand 创建 L10 手部设备实例
func NewL10Hand(config map[string]any) (device.Device, error) {
	id, ok := config["id"].(string)
	if !ok {
		return nil, fmt.Errorf("缺少设备 ID 配置")
	}

	serviceURL, ok := config["can_service_url"].(string)
	if !ok {
		return nil, fmt.Errorf("缺少 can 服务 URL 配置")
	}

	canInterface, ok := config["can_interface"].(string)
	if !ok {
		canInterface = "can0" // 默认接口
	}

	handTypeStr, ok := config["hand_type"].(string)
	handType := define.HAND_TYPE_RIGHT // 默认右手
	if ok && handTypeStr == "left" {
		handType = define.HAND_TYPE_LEFT
	}

	// 创建通信客户端
	comm := communication.NewCanBridgeClient(serviceURL)

	hand := &L10Hand{
		id:           id,
		model:        "L10",
		handType:     handType,
		communicator: comm,
		components:   make(map[device.ComponentType][]device.Component),
		canInterface: canInterface,
		status: device.DeviceStatus{
			IsConnected: false,
			IsActive:    false,
			LastUpdate:  time.Now(),
		},
	}

	// 初始化动画引擎，将 hand 自身作为 PoseExecutor
	hand.animationEngine = device.NewAnimationEngine(hand)

	// 注册默认动画
	hand.animationEngine.Register(NewL10WaveAnimation())
	hand.animationEngine.Register(NewL10SwayAnimation())

	// 初始化组件
	if err := hand.initializeComponents(config); err != nil {
		return nil, fmt.Errorf("初始化组件失败：%w", err)
	}

	log.Printf("✅ 设备 L10 (%s, %s) 创建成功", id, handType.String())
	return hand, nil
}

// GetHandType 获取设备手型
func (h *L10Hand) GetHandType() define.HandType {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return h.handType
}

// SetHandType 设置设备手型
func (h *L10Hand) SetHandType(handType define.HandType) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	if handType != define.HAND_TYPE_LEFT && handType != define.HAND_TYPE_RIGHT {
		return fmt.Errorf("无效的手型：%d", handType)
	}
	h.handType = handType
	log.Printf("🔧 设备 %s 手型已更新: %s", h.id, handType.String())
	return nil
}

// GetAnimationEngine 获取动画引擎
func (h *L10Hand) GetAnimationEngine() *device.AnimationEngine {
	return h.animationEngine
}

// SetFingerPose 设置手指姿态 (实现 PoseExecutor)
func (h *L10Hand) SetFingerPose(pose []byte) error {
	if len(pose) != 6 {
		return fmt.Errorf("无效的手指姿态数据长度，需要 6 个字节")
	}

	// 添加随机扰动
	perturbedPose := make([]byte, len(pose))
	for i, v := range pose {
		perturbedPose[i] = perturb(v, 5)
	}

	// 创建指令
	cmd := device.NewFingerPoseCommand("all", perturbedPose)

	// 执行指令
	err := h.ExecuteCommand(cmd)
	if err == nil {
		log.Printf("✅ %s (%s) 手指动作已发送: [%X %X %X %X %X %X]",
			h.id, h.GetHandType().String(), perturbedPose[0], perturbedPose[1], perturbedPose[2],
			perturbedPose[3], perturbedPose[4], perturbedPose[5])
	}
	return err
}

// SetPalmPose 设置手掌姿态 (实现 PoseExecutor)
func (h *L10Hand) SetPalmPose(pose []byte) error {
	if len(pose) != 4 {
		return fmt.Errorf("无效的手掌姿态数据长度，需要 4 个字节")
	}

	// 添加随机扰动
	perturbedPose := make([]byte, len(pose))
	for i, v := range pose {
		perturbedPose[i] = perturb(v, 8)
	}

	// 创建指令
	cmd := device.NewPalmPoseCommand(perturbedPose)

	// 执行指令
	err := h.ExecuteCommand(cmd)
	if err == nil {
		log.Printf("✅ %s (%s) 掌部姿态已发送: [%X %X %X %X]",
			h.id, h.GetHandType().String(), perturbedPose[0], perturbedPose[1], perturbedPose[2], perturbedPose[3])
	}
	return err
}

// ResetPose 重置到默认姿态 (实现 PoseExecutor)
func (h *L10Hand) ResetPose() error {
	log.Printf("🔄 正在重置设备 %s (%s) 到默认姿态...", h.id, h.GetHandType().String())
	defaultFingerPose := []byte{64, 64, 64, 64, 64, 64} // 0x40 - 半开
	defaultPalmPose := []byte{128, 128, 128, 128}       // 0x80 - 居中

	if err := h.SetFingerPose(defaultFingerPose); err != nil {
		log.Printf("❌ %s 重置手指姿势失败: %v", h.id, err)
		return err
	}
	time.Sleep(20 * time.Millisecond) // 短暂延时
	if err := h.SetPalmPose(defaultPalmPose); err != nil {
		log.Printf("❌ %s 重置掌部姿势失败: %v", h.id, err)
		return err
	}
	log.Printf("✅ 设备 %s 已重置到默认姿态", h.id)
	return nil
}

// commandToRawMessage 将通用指令转换为 L10 特定的 CAN 消息
func (h *L10Hand) commandToRawMessage(cmd device.Command) (communication.RawMessage, error) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	var data []byte
	canID := uint32(h.handType)

	switch cmd.Type() {
	case "SetFingerPose":
		// 添加 0x01 前缀
		data = append([]byte{0x01}, cmd.Payload()...)
		if len(data) > 8 { // CAN 消息数据长度限制
			return communication.RawMessage{}, fmt.Errorf("手指姿态数据过长")
		}
	case "SetPalmPose":
		// 添加 0x04 前缀
		data = append([]byte{0x04}, cmd.Payload()...)
		if len(data) > 8 { // CAN 消息数据长度限制
			return communication.RawMessage{}, fmt.Errorf("手掌姿态数据过长")
		}
	default:
		return communication.RawMessage{}, fmt.Errorf("L10 不支持的指令类型: %s", cmd.Type())
	}

	return communication.RawMessage{
		Interface: h.canInterface,
		ID:        canID,
		Data:      data,
	}, nil
}

// ExecuteCommand 执行一个通用指令
func (h *L10Hand) ExecuteCommand(cmd device.Command) error {
	h.mutex.Lock() // 使用写锁，因为会更新状态
	defer h.mutex.Unlock()

	if !h.status.IsConnected || !h.status.IsActive {
		return fmt.Errorf("设备 %s 未连接或未激活", h.id)
	}

	// 转换指令为 CAN 消息
	rawMsg, err := h.commandToRawMessage(cmd)
	if err != nil {
		h.status.ErrorCount++
		h.status.LastError = err.Error()
		return fmt.Errorf("转换指令失败：%w", err)
	}

	// 发送到 can-bridge 服务
	if err := h.communicator.SendMessage(rawMsg); err != nil {
		h.status.ErrorCount++
		h.status.LastError = err.Error()
		log.Printf("❌ %s (%s) 发送指令失败: %v (ID: 0x%X, Data: %X)", h.id, h.handType.String(), err, rawMsg.ID, rawMsg.Data)
		return fmt.Errorf("发送指令失败：%w", err)
	}

	h.status.LastUpdate = time.Now()
	// 成功的日志记录移到 SetFingerPose 和 SetPalmPose 中，因为那里有更详细的信息
	return nil
}

// --- 其他 L10Hand 方法 (initializeComponents, GetID, GetModel, ReadSensorData, etc.) 保持不变 ---
// --- 确保它们存在且与您上传的版本一致 ---

func (h *L10Hand) initializeComponents(_ map[string]any) error {
	// 初始化传感器组件
	sensors := []device.Component{
		component.NewPressureSensor("pressure_thumb", map[string]any{"location": "thumb"}),
		component.NewPressureSensor("pressure_index", map[string]any{"location": "index"}),
		component.NewPressureSensor("pressure_middle", map[string]any{"location": "middle"}),
		component.NewPressureSensor("pressure_ring", map[string]any{"location": "ring"}),
		component.NewPressureSensor("pressure_pinky", map[string]any{"location": "pinky"}),
	}
	h.components[device.SensorComponent] = sensors
	return nil
}

func (h *L10Hand) GetID() string {
	return h.id
}

func (h *L10Hand) GetModel() string {
	return h.model
}

func (h *L10Hand) ReadSensorData(sensorID string) (device.SensorData, error) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	sensors := h.components[device.SensorComponent]
	for _, comp := range sensors {
		if comp.GetID() == sensorID {
			if sensor, ok := comp.(component.Sensor); ok {
				return sensor.ReadData()
			}
		}
	}
	return nil, fmt.Errorf("传感器 %s 不存在", sensorID)
}

func (h *L10Hand) GetComponents(componentType device.ComponentType) []device.Component {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	if components, exists := h.components[componentType]; exists {
		result := make([]device.Component, len(components))
		copy(result, components)
		return result
	}
	return []device.Component{}
}

func (h *L10Hand) GetStatus() (device.DeviceStatus, error) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return h.status, nil
}

func (h *L10Hand) Connect() error {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	// TODO: 假设连接总是成功，除非有显式错误
	h.status.IsConnected = true
	h.status.IsActive = true
	h.status.LastUpdate = time.Now()
	log.Printf("🔗 设备 %s 已连接", h.id)
	return nil
}

func (h *L10Hand) Disconnect() error {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.status.IsConnected = false
	h.status.IsActive = false
	h.status.LastUpdate = time.Now()
	log.Printf("🔌 设备 %s 已断开", h.id)
	return nil
}
