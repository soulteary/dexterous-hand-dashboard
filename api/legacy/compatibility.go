package legacy

import (
	"fmt"
	"log"
	"sync"

	"hands/config"
	"hands/define"
	"hands/device"
)

// InterfaceDeviceMapper 管理接口和设备的映射关系
type InterfaceDeviceMapper struct {
	interfaceToDevice map[string]string     // interface -> deviceId
	deviceToInterface map[string]string     // deviceId -> interface
	handConfigs       map[string]HandConfig // interface -> hand config
	deviceManager     *device.DeviceManager
	mutex             sync.RWMutex
}

// HandConfig 存储接口的手型配置（兼容旧版 API）
type HandConfig struct {
	HandType string
	HandId   uint32
}

// NewInterfaceDeviceMapper 创建新的接口设备映射器
func NewInterfaceDeviceMapper(deviceManager *device.DeviceManager) (*InterfaceDeviceMapper, error) {
	mapper := &InterfaceDeviceMapper{
		interfaceToDevice: make(map[string]string),
		deviceToInterface: make(map[string]string),
		handConfigs:       make(map[string]HandConfig),
		deviceManager:     deviceManager,
	}

	if err := mapper.initializeDevices(); err != nil {
		return nil, fmt.Errorf("初始化设备映射失败：%w", err)
	}

	return mapper, nil
}

// initializeDevices 为每个可用接口创建对应的设备实例
func (m *InterfaceDeviceMapper) initializeDevices() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	log.Printf("🔧 开始为 %d 个接口创建设备映射...", len(config.Config.AvailableInterfaces))

	for _, ifName := range config.Config.AvailableInterfaces {
		deviceId := ifName + "_default"

		// 创建设备配置
		deviceConfig := map[string]any{
			"id":              deviceId,
			"can_service_url": config.Config.CanServiceURL,
			"can_interface":   ifName,
			"hand_type":       "right", // 默认右手
		}

		// 创建设备实例
		dev, err := device.CreateDevice("L10", deviceConfig)
		if err != nil {
			return fmt.Errorf("创建接口 %s 的设备失败: %w", ifName, err)
		}

		// 注册设备到管理器
		if err := m.deviceManager.RegisterDevice(dev); err != nil {
			return fmt.Errorf("注册接口 %s 的设备失败: %w", ifName, err)
		}

		// 建立映射关系
		m.interfaceToDevice[ifName] = deviceId
		m.deviceToInterface[deviceId] = ifName

		// 初始化手型配置
		m.handConfigs[ifName] = HandConfig{
			HandType: "right",
			HandId:   uint32(define.HAND_TYPE_RIGHT),
		}

		log.Printf("✅ 接口 %s -> 设备 %s 映射创建成功", ifName, deviceId)
	}

	log.Printf("🎉 设备映射初始化完成，共创建 %d 个设备", len(config.Config.AvailableInterfaces))
	return nil
}

// GetDeviceForInterface 根据接口名获取对应的设备
func (m *InterfaceDeviceMapper) GetDeviceForInterface(ifName string) (device.Device, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	deviceId, exists := m.interfaceToDevice[ifName]
	if !exists {
		return nil, fmt.Errorf("接口 %s 没有对应的设备", ifName)
	}

	return m.deviceManager.GetDevice(deviceId)
}

// GetInterfaceForDevice 根据设备 ID 获取对应的接口名
func (m *InterfaceDeviceMapper) GetInterfaceForDevice(deviceId string) (string, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	ifName, exists := m.deviceToInterface[deviceId]
	return ifName, exists
}

// SetHandConfig 设置接口的手型配置
func (m *InterfaceDeviceMapper) SetHandConfig(ifName string, handType string, handId uint32) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// 验证接口是否存在
	if !config.IsValidInterface(ifName) {
		return fmt.Errorf("无效的接口: %s", ifName)
	}

	// 获取对应的设备
	deviceId, exists := m.interfaceToDevice[ifName]
	if !exists {
		return fmt.Errorf("接口 %s 没有对应的设备", ifName)
	}

	dev, err := m.deviceManager.GetDevice(deviceId)
	if err != nil {
		return fmt.Errorf("获取设备失败：%w", err)
	}

	// 转换手型
	var deviceHandType define.HandType
	switch handType {
	case "left":
		deviceHandType = define.HAND_TYPE_LEFT
	case "right":
		deviceHandType = define.HAND_TYPE_RIGHT
	default:
		return fmt.Errorf("无效的手型: %s", handType)
	}

	// 设置设备手型
	if err := dev.SetHandType(deviceHandType); err != nil {
		return fmt.Errorf("设置设备手型失败：%w", err)
	}

	// 更新本地配置
	m.handConfigs[ifName] = HandConfig{
		HandType: handType,
		HandId:   handId,
	}

	log.Printf("🔧 接口 %s 手型已设置为 %s (0x%X)", ifName, handType, handId)
	return nil
}

// GetHandConfig 获取接口的手型配置
func (m *InterfaceDeviceMapper) GetHandConfig(ifName string) (HandConfig, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	config, exists := m.handConfigs[ifName]
	return config, exists
}

// GetAllHandConfigs 获取所有接口的手型配置
func (m *InterfaceDeviceMapper) GetAllHandConfigs() map[string]HandConfig {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	result := make(map[string]HandConfig)
	for ifName, config := range m.handConfigs {
		result[ifName] = config
	}
	return result
}

// StopAllAnimations 停止指定接口对应设备的动画
func (m *InterfaceDeviceMapper) StopAllAnimations(ifName string) error {
	dev, err := m.GetDeviceForInterface(ifName)
	if err != nil {
		return err
	}

	animEngine := dev.GetAnimationEngine()
	if animEngine.IsRunning() {
		return animEngine.Stop()
	}
	return nil
}

// GetDeviceStatus 获取指定接口对应设备的状态
func (m *InterfaceDeviceMapper) GetDeviceStatus(ifName string) (device.DeviceStatus, error) {
	dev, err := m.GetDeviceForInterface(ifName)
	if err != nil {
		return device.DeviceStatus{}, err
	}

	return dev.GetStatus()
}

// IsValidInterface 验证接口是否有效
func (m *InterfaceDeviceMapper) IsValidInterface(ifName string) bool {
	return config.IsValidInterface(ifName)
}
