package device

import (
	"fmt"
	"sync"
)

// DeviceManager 管理设备实例
type DeviceManager struct {
	devices map[string]Device
	mutex   sync.RWMutex
}

func NewDeviceManager() *DeviceManager { return &DeviceManager{devices: make(map[string]Device)} }

func (m *DeviceManager) RegisterDevice(dev Device) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	id := dev.GetID()
	if _, exists := m.devices[id]; exists {
		return fmt.Errorf("设备 %s 已存在", id)
	}

	m.devices[id] = dev
	return nil
}

func (m *DeviceManager) GetDevice(id string) (Device, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	dev, exists := m.devices[id]
	if !exists {
		return nil, fmt.Errorf("设备 %s 不存在", id)
	}

	return dev, nil
}

func (m *DeviceManager) GetAllDevices() []Device {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	devices := make([]Device, 0, len(m.devices))
	for _, dev := range m.devices {
		devices = append(devices, dev)
	}

	return devices
}

func (m *DeviceManager) RemoveDevice(id string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.devices[id]; !exists {
		return fmt.Errorf("设备 %s 不存在", id)
	}

	delete(m.devices, id)
	return nil
}
