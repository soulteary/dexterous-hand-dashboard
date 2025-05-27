package component

import (
	"hands/pkg/device"
	"time"
)

// Sensor 传感器组件接口
type Sensor interface {
	device.Component
	ReadData() (device.SensorData, error)
	GetDataType() string
	GetSamplingRate() int
	SetSamplingRate(rate int) error
}

// SensorDataImpl 传感器数据的具体实现
type SensorDataImpl struct {
	timestamp time.Time
	values    map[string]any
	sensorID  string
}

func NewSensorData(sensorID string, values map[string]any) *SensorDataImpl {
	return &SensorDataImpl{
		timestamp: time.Now(),
		values:    values,
		sensorID:  sensorID,
	}
}

func (s *SensorDataImpl) Timestamp() time.Time {
	return s.timestamp
}

func (s *SensorDataImpl) Values() map[string]any {
	return s.values
}

func (s *SensorDataImpl) SensorID() string {
	return s.sensorID
}
