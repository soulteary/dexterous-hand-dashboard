package device

import (
	"hands/define"
	"time"
)

// Device 代表一个可控制的设备单元
type Device interface {
	GetID() string                                         // 获取设备唯一标识
	GetModel() string                                      // 获取设备型号 (例如 "L10", "L20")
	GetHandType() define.HandType                          // 获取设备手型
	SetHandType(handType define.HandType) error            // 设置设备手型
	ExecuteCommand(cmd Command) error                      // 执行一个通用指令
	ReadSensorData(sensorID string) (SensorData, error)    // 读取特定传感器数据
	GetComponents(componentType ComponentType) []Component // 获取指定类型的组件
	GetStatus() (DeviceStatus, error)                      // 获取设备状态
	Connect() error                                        // 连接设备
	Disconnect() error                                     // 断开设备连接

	// --- 新增 ---
	PoseExecutor                          // 嵌入 PoseExecutor 接口，Device 需实现它
	GetAnimationEngine() *AnimationEngine // 获取设备的动画引擎

	// --- 预设姿势相关方法 ---
	GetSupportedPresets() []string                 // 获取支持的预设姿势列表
	ExecutePreset(presetName string) error         // 执行预设姿势
	GetPresetDescription(presetName string) string // 获取预设姿势描述
}

// Command 代表一个发送给设备的指令
type Command interface {
	Type() string            // 指令类型，例如 "SetFingerPose", "SetPalmAngle"
	Payload() []byte         // 指令的实际数据
	TargetComponent() string // 目标组件 ID
}

// SensorData 代表从传感器读取的数据
type SensorData interface {
	Timestamp() time.Time
	Values() map[string]any // 例如 {"pressure": 100, "angle": 30.5}
	SensorID() string
}

// ComponentType 定义组件类型
type ComponentType string

const (
	SensorComponent   ComponentType = "sensor"
	SkinComponent     ComponentType = "skin"
	ActuatorComponent ComponentType = "actuator"
)

// Component 代表设备的一个可插拔组件
type Component interface {
	GetID() string
	GetType() ComponentType
	GetConfiguration() map[string]interface{} // 组件的特定配置
	IsActive() bool
}

// DeviceStatus 代表设备状态
type DeviceStatus struct {
	IsConnected bool
	IsActive    bool
	LastUpdate  time.Time
	ErrorCount  int
	LastError   string
}
