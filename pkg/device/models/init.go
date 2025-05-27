package models

import "hands/pkg/device"

func init() {
	// 注册 L10 设备类型
	device.RegisterDeviceType("L10", NewL10Hand)
}
