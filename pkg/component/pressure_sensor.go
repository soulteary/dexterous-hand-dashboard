package component

import (
	"fmt"
	"hands/pkg/device"
	"math/rand/v2"
	"time"
)

// PressureSensor 压力传感器实现
type PressureSensor struct {
	id           string
	config       map[string]any
	isActive     bool
	samplingRate int
	lastReading  time.Time
}

func NewPressureSensor(id string, config map[string]any) *PressureSensor {
	return &PressureSensor{
		id:           id,
		config:       config,
		isActive:     true,
		samplingRate: 100,
		lastReading:  time.Now(),
	}
}

func (p *PressureSensor) GetID() string {
	return p.id
}

func (p *PressureSensor) GetType() device.ComponentType {
	return device.SensorComponent
}

func (p *PressureSensor) GetConfiguration() map[string]any {
	return p.config
}

func (p *PressureSensor) IsActive() bool {
	return p.isActive
}

func (p *PressureSensor) ReadData() (device.SensorData, error) {
	if !p.isActive {
		return nil, fmt.Errorf("传感器 %s 未激活", p.id)
	}

	// 模拟压力数据读取
	// 在实际实现中，这里应该从 can-bridge 或其他数据源读取真实数据
	pressure := rand.Float64() * 100 // 0-100 的随机压力值

	values := map[string]any{
		"pressure": pressure,
		"unit":     "kPa",
		"location": p.config["location"],
	}

	p.lastReading = time.Now()
	return NewSensorData(p.id, values), nil
}

func (p *PressureSensor) GetDataType() string {
	return "pressure"
}

func (p *PressureSensor) GetSamplingRate() int {
	return p.samplingRate
}

func (p *PressureSensor) SetSamplingRate(rate int) error {
	if rate <= 0 || rate > 1000 {
		return fmt.Errorf("采样率必须在 1-1000Hz 之间")
	}
	p.samplingRate = rate
	return nil
}
