package hands

import (
	"hands/config"
	"log"
	"sync"
	"time"
)

var (
	AnimationActive  map[string]bool // 每个接口的动画状态
	AnimationMutex   sync.Mutex
	StopAnimationMap map[string]chan struct{} // 每个接口的停止动画通道
)

func initAnimation() {
	// 初始化动画状态映射
	AnimationActive = make(map[string]bool)
	StopAnimationMap = make(map[string]chan struct{})
	for _, ifName := range config.Config.AvailableInterfaces {
		AnimationActive[ifName] = false
		StopAnimationMap[ifName] = make(chan struct{}, 1)
	}
}

// 执行波浪动画 - 支持手型参数
func StartWaveAnimation(ifName string, speed int, handType string, handId uint32) {
	if speed <= 0 {
		speed = 500 // 默认速度
	}

	// 如果未指定接口，使用默认接口
	if ifName == "" {
		ifName = config.Config.DefaultInterface
	}

	// 验证接口
	if !config.IsValidInterface(ifName) {
		log.Printf("❌ 无法启动波浪动画: 无效的接口 %s", ifName)
		return
	}

	AnimationMutex.Lock()

	// 如果已经有动画在运行，先停止它
	if AnimationActive[ifName] {
		select {
		case StopAnimationMap[ifName] <- struct{}{}:
			// 发送成功
		default:
			// 通道已满，无需发送
		}

		StopAnimationMap[ifName] = make(chan struct{}, 1)
	}

	AnimationActive[ifName] = true
	AnimationMutex.Unlock()

	currentStopChannel := StopAnimationMap[ifName]

	go func() {
		defer func() {
			AnimationMutex.Lock()
			AnimationActive[ifName] = false
			AnimationMutex.Unlock()
			log.Printf("👋 %s 波浪动画已完成", ifName)
		}()

		fingerOrder := []int{0, 1, 2, 3, 4, 5}
		open := byte(64)   // 0x40
		close := byte(192) // 0xC0

		log.Printf("🚀 开始 %s 波浪动画", ifName)

		// 动画循环
		for {
			select {
			case <-currentStopChannel:
				log.Printf("🛑 %s 波浪动画被用户停止", ifName)
				return
			default:
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

					if err := SendFingerPose(ifName, pose, handType, handId); err != nil {
						log.Printf("%s 动画发送失败: %v", ifName, err)
						return
					}

					delay := time.Duration(speed) * time.Millisecond

					select {
					case <-currentStopChannel:
						log.Printf("🛑 %s 波浪动画被用户停止", ifName)
						return
					case <-time.After(delay):
						// 继续执行
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

					if err := SendFingerPose(ifName, pose, handType, handId); err != nil {
						log.Printf("%s 动画发送失败: %v", ifName, err)
						return
					}

					delay := time.Duration(speed) * time.Millisecond

					select {
					case <-currentStopChannel:
						log.Printf("🛑 %s 波浪动画被用户停止", ifName)
						return
					case <-time.After(delay):
						// 继续执行
					}
				}
			}
		}
	}()
}

// 执行横向摆动动画 - 支持手型参数
func StartSwayAnimation(ifName string, speed int, handType string, handId uint32) {
	if speed <= 0 {
		speed = 500 // 默认速度
	}

	// 如果未指定接口，使用默认接口
	if ifName == "" {
		ifName = config.Config.DefaultInterface
	}

	// 验证接口
	if !config.IsValidInterface(ifName) {
		log.Printf("❌ 无法启动摆动动画: 无效的接口 %s", ifName)
		return
	}

	AnimationMutex.Lock()

	if AnimationActive[ifName] {
		select {
		case StopAnimationMap[ifName] <- struct{}{}:
			// 发送成功
		default:
			// 通道已满，无需发送
		}

		StopAnimationMap[ifName] = make(chan struct{}, 1)
	}

	AnimationActive[ifName] = true
	AnimationMutex.Unlock()

	currentStopChannel := StopAnimationMap[ifName]

	go func() {
		defer func() {
			AnimationMutex.Lock()
			AnimationActive[ifName] = false
			AnimationMutex.Unlock()
			log.Printf("🔄 %s 横向摆动动画已完成", ifName)
		}()

		leftPose := []byte{48, 48, 48, 48}      // 0x30
		rightPose := []byte{208, 208, 208, 208} // 0xD0

		log.Printf("🚀 开始 %s 横向摆动动画", ifName)

		// 动画循环
		for {
			select {
			case <-currentStopChannel:
				log.Printf("🛑 %s 横向摆动动画被用户停止", ifName)
				return
			default:
				// 向左移动
				if err := SendPalmPose(ifName, leftPose, handType, handId); err != nil {
					log.Printf("%s 动画发送失败: %v", ifName, err)
					return
				}

				delay := time.Duration(speed) * time.Millisecond

				select {
				case <-currentStopChannel:
					log.Printf("🛑 %s 横向摆动动画被用户停止", ifName)
					return
				case <-time.After(delay):
					// 继续执行
				}

				// 向右移动
				if err := SendPalmPose(ifName, rightPose, handType, handId); err != nil {
					log.Printf("%s 动画发送失败: %v", ifName, err)
					return
				}

				select {
				case <-currentStopChannel:
					log.Printf("🛑 %s 横向摆动动画被用户停止", ifName)
					return
				case <-time.After(delay):
					// 继续执行
				}
			}
		}
	}()
}

// 停止所有动画
func StopAllAnimations(ifName string) {
	// 如果未指定接口，停止所有接口的动画
	if ifName == "" {
		for _, validIface := range config.Config.AvailableInterfaces {
			StopAllAnimations(validIface)
		}
		return
	}

	// 验证接口
	if !config.IsValidInterface(ifName) {
		log.Printf("⚠️ 尝试停止无效接口的动画: %s", ifName)
		return
	}

	AnimationMutex.Lock()
	defer AnimationMutex.Unlock()

	if AnimationActive[ifName] {
		select {
		case StopAnimationMap[ifName] <- struct{}{}:
			log.Printf("✅ 已发送停止 %s 动画信号", ifName)
		default:
			StopAnimationMap[ifName] = make(chan struct{}, 1)
			StopAnimationMap[ifName] <- struct{}{}
			log.Printf("⚠️ %s 通道重置后发送了停止信号", ifName)
		}

		AnimationActive[ifName] = false

		go func() {
			time.Sleep(100 * time.Millisecond)
			resetToDefaultPose(ifName)
		}()
	} else {
		log.Printf("ℹ️ %s 当前没有运行中的动画", ifName)
	}
}
