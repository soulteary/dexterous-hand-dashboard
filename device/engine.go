package device

import (
	"fmt"
	"log"
	"sync"
)

// defaultAnimationSpeedMs 定义默认动画速度（毫秒）
const defaultAnimationSpeedMs = 500

// AnimationEngine 管理和执行动画
type AnimationEngine struct {
	executor      PoseExecutor         // 关联的姿态执行器
	animations    map[string]Animation // 注册的动画
	stopChan      chan struct{}        // 当前动画的停止通道
	current       string               // 当前运行的动画名称
	isRunning     bool                 // 是否有动画在运行
	engineMutex   sync.Mutex           // 保护引擎状态 (isRunning, current, stopChan)
	registerMutex sync.RWMutex         // 保护动画注册表 (animations)
}

// NewAnimationEngine 创建一个新的动画引擎
func NewAnimationEngine(executor PoseExecutor) *AnimationEngine {
	return &AnimationEngine{
		executor:   executor,
		animations: make(map[string]Animation),
	}
}

// Register 注册一个动画
func (e *AnimationEngine) Register(anim Animation) {
	e.registerMutex.Lock()
	defer e.registerMutex.Unlock()

	if anim == nil {
		log.Printf("⚠️ 尝试注册一个空动画")
		return
	}

	name := anim.Name()
	if _, exists := e.animations[name]; exists {
		log.Printf("⚠️ 动画 %s 已注册，将被覆盖", name)
	}
	e.animations[name] = anim
	log.Printf("✅ 动画 %s 已注册", name)
}

// getAnimation 安全地获取一个已注册的动画
func (e *AnimationEngine) getAnimation(name string) (Animation, bool) {
	e.registerMutex.RLock()
	defer e.registerMutex.RUnlock()
	anim, exists := e.animations[name]
	return anim, exists
}

// getDeviceName 尝试获取设备 ID 用于日志记录
func (e *AnimationEngine) getDeviceName() string {
	// 尝试通过接口断言获取 ID
	if idProvider, ok := e.executor.(interface{ GetID() string }); ok {
		return idProvider.GetID()
	}
	return "设备" // 默认名称
}

// Start 启动一个动画
func (e *AnimationEngine) Start(name string, speedMs int) error {
	e.engineMutex.Lock()
	defer e.engineMutex.Unlock() // 确保在任何情况下都释放锁

	anim, exists := e.getAnimation(name)
	if !exists {
		return fmt.Errorf("❌ 动画 %s 未注册", name)
	}

	// 如果有动画在运行，先发送停止信号
	if e.isRunning {
		log.Printf("ℹ️ 正在停止当前动画 %s 以启动 %s...", e.current, name)
		close(e.stopChan)
		// 注意：我们不在此处等待旧动画结束。
		// 新动画将立即启动，旧动画的 goroutine 在收到信号后会退出。
		// 其 defer 中的 `stopChan` 比较会确保它不会干扰新动画的状态。
	}

	// 设置新动画状态
	e.stopChan = make(chan struct{}) // 创建新的停止通道
	e.isRunning = true
	e.current = name

	// 验证并设置速度
	actualSpeedMs := speedMs
	if actualSpeedMs <= 0 {
		actualSpeedMs = defaultAnimationSpeedMs
	}

	log.Printf("🚀 准备启动动画 %s (设备: %s, 速度: %dms)", name, e.getDeviceName(), actualSpeedMs)

	// 启动动画 goroutine
	go e.runAnimationLoop(anim, e.stopChan, actualSpeedMs)

	return nil
}

// Stop 停止当前正在运行的动画
func (e *AnimationEngine) Stop() error {
	e.engineMutex.Lock()
	defer e.engineMutex.Unlock()

	if !e.isRunning {
		log.Printf("ℹ️ 当前没有动画在运行 (设备: %s)", e.getDeviceName())
		return nil
	}

	log.Printf("⏳ 正在发送停止信号给动画 %s (设备: %s)...", e.current, e.getDeviceName())
	close(e.stopChan)   // 发送停止信号
	e.isRunning = false // 立即标记为未运行，防止重复停止
	e.current = ""
	// 动画的 goroutine 将在下一次检查通道时退出，
	// 并在其 defer 块中执行最终的清理（包括 ResetPose）。

	return nil
}

// IsRunning 检查是否有动画在运行
func (e *AnimationEngine) IsRunning() bool {
	e.engineMutex.Lock()
	defer e.engineMutex.Unlock()
	return e.isRunning
}

// GetRegisteredAnimations 获取已注册的动画名称列表
func (e *AnimationEngine) GetRegisteredAnimations() []string {
	e.registerMutex.RLock()
	defer e.registerMutex.RUnlock()

	animations := make([]string, 0, len(e.animations))
	for name := range e.animations {
		animations = append(animations, name)
	}
	return animations
}

// GetCurrentAnimation 获取当前运行的动画名称
func (e *AnimationEngine) GetCurrentAnimation() string {
	e.engineMutex.Lock()
	defer e.engineMutex.Unlock()
	return e.current
}

// runAnimationLoop 是动画执行的核心循环，在单独的 Goroutine 中运行。
func (e *AnimationEngine) runAnimationLoop(anim Animation, stopChan <-chan struct{}, speedMs int) {
	deviceName := e.getDeviceName()
	animName := anim.Name()

	// 使用 defer 确保无论如何都能执行清理逻辑
	defer e.handleLoopExit(stopChan, deviceName, animName)

	log.Printf("▶️ %s 动画 %s 已启动", deviceName, animName)

	// 动画主循环
	for {
		select {
		case <-stopChan:
			log.Printf("🛑 %s 动画 %s 被显式停止", deviceName, animName)
			return // 接收到停止信号，退出循环
		default:
			// 执行一轮动画
			err := anim.Run(e.executor, stopChan, speedMs)
			if err != nil {
				log.Printf("❌ %s 动画 %s 执行出错: %v", deviceName, animName, err)
				return // 出错则退出
			}

			// 再次检查停止信号，防止 Run 结束后才收到信号
			select {
			case <-stopChan:
				log.Printf("🛑 %s 动画 %s 在周期结束时被停止", deviceName, animName)
				return
			default:
				// 继续下一个循环
			}
		}
	}
}

// handleLoopExit 是动画 Goroutine 退出时执行的清理函数。
func (e *AnimationEngine) handleLoopExit(stopChan <-chan struct{}, deviceName, animName string) {
	e.engineMutex.Lock()
	defer e.engineMutex.Unlock()

	// --- 关键并发控制 ---
	// 检查当前引擎的 stopChan 是否与此 Goroutine 启动时的 stopChan 相同。
	// 如果不相同，说明一个新的动画已经启动，并且接管了引擎状态。
	// 这种情况下，旧的 Goroutine 不应该修改引擎状态或重置姿态，
	// 以避免干扰新动画。
	if stopChan == e.stopChan {
		// 只有当自己仍然是"活跃"的动画时，才更新状态并重置姿态
		e.isRunning = false
		e.current = ""
		log.Printf("👋 %s 动画 %s 已完成或停止，正在重置姿态...", deviceName, animName)
		if err := e.executor.ResetPose(); err != nil {
			log.Printf("⚠️ %s 动画结束后重置姿态失败: %v", deviceName, err)
		} else {
			log.Printf("✅ %s 姿态已重置", deviceName)
		}
	} else {
		// 如果 stopChan 不同，说明自己是旧的 Goroutine，只需安静退出
		log.Printf("ℹ️ 旧的 %s 动画 %s goroutine 退出，但新动画已启动，无需重置。", deviceName, animName)
	}
}
