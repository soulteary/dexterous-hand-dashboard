package models

import (
	"hands/device"
	"log"
	"time"
)

// --- L10WaveAnimation ---

// L10WaveAnimation 实现 L10 的波浪动画
type L10WaveAnimation struct{}

// NewL10WaveAnimation 创建 L10 波浪动画实例
func NewL10WaveAnimation() *L10WaveAnimation { return &L10WaveAnimation{} }

func (w *L10WaveAnimation) Name() string { return "wave" }

func (w *L10WaveAnimation) Run(executor device.PoseExecutor, stop <-chan struct{}, speedMs int) error {
	fingerOrder := []int{0, 1, 2, 3, 4, 5}
	open := byte(64)   // 0x40
	close := byte(192) // 0xC0
	delay := time.Duration(speedMs) * time.Millisecond

	deviceName := "L10"

	// 波浪张开
	for _, idx := range fingerOrder {
		pose := make([]byte, 6)
		for j := 0; j < 6; j++ {
			if j == idx {
				pose[j] = open
			} else {
				pose[j] = close
			}
		}

		if err := executor.SetFingerPose(pose); err != nil {
			log.Printf("❌ %s 动画 %s 发送失败: %v", deviceName, w.Name(), err)
			return err
		}

		select {
		case <-stop:
			return nil // 动画被停止
		case <-time.After(delay):
			// 继续
		}
	}

	// 波浪握拳
	for _, idx := range fingerOrder {
		pose := make([]byte, 6)
		for j := 0; j < 6; j++ {
			if j == idx {
				pose[j] = close
			} else {
				pose[j] = open
			}
		}

		if err := executor.SetFingerPose(pose); err != nil {
			log.Printf("❌ %s 动画 %s 发送失败: %v", deviceName, w.Name(), err)
			return err
		}

		select {
		case <-stop:
			return nil // 动画被停止
		case <-time.After(delay):
			// 继续
		}
	}

	return nil // 完成一个周期
}

// --- L10SwayAnimation ---

// L10SwayAnimation 实现 L10 的横向摆动动画
type L10SwayAnimation struct{}

// NewL10SwayAnimation 创建 L10 摆动动画实例
func NewL10SwayAnimation() *L10SwayAnimation { return &L10SwayAnimation{} }

func (s *L10SwayAnimation) Name() string { return "sway" }

func (s *L10SwayAnimation) Run(executor device.PoseExecutor, stop <-chan struct{}, speedMs int) error {
	leftPose := []byte{48, 48, 48, 48}      // 0x30
	rightPose := []byte{208, 208, 208, 208} // 0xD0
	delay := time.Duration(speedMs) * time.Millisecond

	deviceName := "L10"
	if idProvider, ok := executor.(interface{ GetID() string }); ok {
		deviceName = idProvider.GetID()
	}

	// 向左移动
	if err := executor.SetPalmPose(leftPose); err != nil {
		log.Printf("❌ %s 动画 %s 发送失败: %v", deviceName, s.Name(), err)
		return err
	}

	select {
	case <-stop:
		return nil // 动画被停止
	case <-time.After(delay):
		// 继续
	}

	// 向右移动
	if err := executor.SetPalmPose(rightPose); err != nil {
		log.Printf("❌ %s 动画 %s 发送失败: %v", deviceName, s.Name(), err)
		return err
	}

	select {
	case <-stop:
		return nil // 动画被停止
	case <-time.After(delay):
		// 继续
	}

	return nil // 完成一个周期
}
