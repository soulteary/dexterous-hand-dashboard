package models

import (
	"fmt"
	"sync"
	"time"

	"hands/define"
	"hands/pkg/communication"
	"hands/pkg/component"
	"hands/pkg/device"
)

// L10Hand L10 型号手部设备实现
type L10Hand struct {
	id           string
	model        string
	handType     define.HandType
	communicator communication.Communicator
	components   map[device.ComponentType][]device.Component
	status       device.DeviceStatus
	mutex        sync.RWMutex
	canInterface string // CAN 接口名称，如 "can0"
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

	// 初始化组件
	if err := hand.initializeComponents(config); err != nil {
		return nil, fmt.Errorf("初始化组件失败：%w", err)
	}

	return hand, nil
}

func (h *L10Hand) GetHandType() define.HandType {
	return h.handType
}

func (h *L10Hand) SetHandType(handType define.HandType) error {
	h.handType = handType
	return nil
}

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

func (h *L10Hand) ExecuteCommand(cmd device.Command) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	// 将通用指令转换为 L10 特定的 CAN 消息
	rawMsg, err := h.commandToRawMessage(cmd)
	if err != nil {
		return fmt.Errorf("转换指令失败：%w", err)
	}

	// 发送到 can-bridge 服务
	if err := h.communicator.SendMessage(rawMsg); err != nil {
		h.status.ErrorCount++
		h.status.LastError = err.Error()
		return fmt.Errorf("发送指令失败：%w", err)
	}

	h.status.LastUpdate = time.Now()
	return nil
}

func (h *L10Hand) commandToRawMessage(cmd device.Command) (communication.RawMessage, error) {
	var canID uint32
	var data []byte

	switch cmd.Type() {
	case "SetFingerPose":
		// 根据目标组件确定 CAN ID
		canID = h.getFingerCanID(cmd.TargetComponent())
		data = cmd.Payload()
	case "SetPalmPose":
		canID = h.getPalmCanID()
		data = cmd.Payload()
	default:
		return communication.RawMessage{}, fmt.Errorf("不支持的指令类型: %s", cmd.Type())
	}

	return communication.RawMessage{
		Interface: h.canInterface,
		ID:        canID,
		Data:      data,
	}, nil
}

func (h *L10Hand) getFingerCanID(targetComponent string) uint32 {
	// L10 设备的手指 CAN ID 映射
	fingerIDs := map[string]uint32{
		"finger_thumb":  0x100,
		"finger_index":  0x101,
		"finger_middle": 0x102,
		"finger_ring":   0x103,
		"finger_pinky":  0x104,
	}

	if id, exists := fingerIDs[targetComponent]; exists {
		return id
	}
	return 0x100 // 默认拇指
}

func (h *L10Hand) getPalmCanID() uint32 {
	return 0x200 // L10 设备的手掌 CAN ID
}

func (h *L10Hand) ReadSensorData(sensorID string) (device.SensorData, error) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	// 查找传感器组件
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

	// 检查与 can-bridge 服务的连接
	if !h.communicator.IsConnected() {
		return fmt.Errorf("无法连接到 can-bridge 服务")
	}

	// 检查 CAN 接口状态
	isActive, err := h.communicator.GetInterfaceStatus(h.canInterface)
	if err != nil {
		return fmt.Errorf("检查 CAN 接口状态失败：%w", err)
	}

	if !isActive {
		return fmt.Errorf("CAN接口 %s 未激活", h.canInterface)
	}

	h.status.IsConnected = true
	h.status.IsActive = true
	h.status.LastUpdate = time.Now()

	return nil
}

func (h *L10Hand) Disconnect() error {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.status.IsConnected = false
	h.status.IsActive = false
	h.status.LastUpdate = time.Now()

	return nil
}
