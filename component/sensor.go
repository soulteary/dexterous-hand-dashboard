package component

import (
	"hands/device"
	"math/rand/v2"
	"time"
)

// Sensor 传感器组件接口
type Sensor interface {
	// device.Component
	ReadData() (device.SensorData, error)
	// GetDataType() string
	// GetSamplingRate() int
	// SetSamplingRate(rate int) error
	MockData()
}

// SensorDataImpl 传感器数据的具体实现
type SensorDataImpl struct {
	Interface    string    `json:"interface"`
	Thumb        int       `json:"thumb"`
	Index        int       `json:"index"`
	Middle       int       `json:"middle"`
	Ring         int       `json:"ring"`
	Pinky        int       `json:"pinky"`
	PalmPosition []byte    `json:"palmPosition"`
	LastUpdate   time.Time `json:"lastUpdate"`
}

func NewSensorData(ifName string) *SensorDataImpl {
	return &SensorDataImpl{
		Interface:    ifName,
		Thumb:        0,
		Index:        0,
		Middle:       0,
		Ring:         0,
		Pinky:        0,
		PalmPosition: []byte{128, 128, 128, 128},
		LastUpdate:   time.Now(),
	}
}

func (s *SensorDataImpl) MockData() {
	go func() {
		for {
			s.Thumb = rand.IntN(101)
			s.Index = rand.IntN(101)
			s.Middle = rand.IntN(101)
			s.Ring = rand.IntN(101)
			s.Pinky = rand.IntN(101)
			s.LastUpdate = time.Now()
			time.Sleep(500 * time.Millisecond)
		}
	}()
}

func (s *SensorDataImpl) Values() map[string]any {
	return map[string]any{
		"thumb":        s.Thumb,
		"index":        s.Index,
		"middle":       s.Middle,
		"ring":         s.Ring,
		"pinky":        s.Pinky,
		"palmPosition": s.PalmPosition,
		"lastUpdate":   s.LastUpdate,
	}
}

func (s *SensorDataImpl) ReadData() (device.SensorData, error) { return s, nil }
