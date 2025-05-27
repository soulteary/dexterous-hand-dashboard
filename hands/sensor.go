package hands

import (
	"hands/config"
	"math/rand/v2"
	"sync"
	"time"
)

var (
	SensorDataMap map[string]*SensorData // 每个接口的传感器数据
	SensorMutex   sync.RWMutex
)

func initSensorData() {
	// 初始化传感器数据映射
	SensorDataMap = make(map[string]*SensorData)
	for _, ifName := range config.Config.AvailableInterfaces {
		SensorDataMap[ifName] = &SensorData{
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
}

// 读取传感器数据 (模拟)
func ReadSensorData() {
	go func() {
		for {
			SensorMutex.Lock()
			// 为每个接口模拟压力数据 (0-100)
			for _, ifName := range config.Config.AvailableInterfaces {
				if sensorData, exists := SensorDataMap[ifName]; exists {
					sensorData.Thumb = rand.IntN(101)
					sensorData.Index = rand.IntN(101)
					sensorData.Middle = rand.IntN(101)
					sensorData.Ring = rand.IntN(101)
					sensorData.Pinky = rand.IntN(101)
					sensorData.LastUpdate = time.Now()
				}
			}
			SensorMutex.Unlock()

			time.Sleep(500 * time.Millisecond)
		}
	}()
}
