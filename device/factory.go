package device

import "fmt"

// DeviceFactory 设备工厂
type DeviceFactory struct {
	constructors map[string]func(config map[string]any) (Device, error)
}

var defaultFactory = &DeviceFactory{
	constructors: make(map[string]func(config map[string]any) (Device, error)),
}

// RegisterDeviceType 注册设备类型
func RegisterDeviceType(modelName string, constructor func(config map[string]any) (Device, error)) {
	defaultFactory.constructors[modelName] = constructor
}

// CreateDevice 创建设备实例
func CreateDevice(modelName string, config map[string]any) (Device, error) {
	constructor, ok := defaultFactory.constructors[modelName]
	if !ok {
		return nil, fmt.Errorf("未知的设备型号: %s", modelName)
	}
	return constructor(config)
}

// GetSupportedModels 获取支持的设备型号列表
func GetSupportedModels() []string {
	models := make([]string, 0, len(defaultFactory.constructors))
	for model := range defaultFactory.constructors {
		models = append(models, model)
	}
	return models
}
