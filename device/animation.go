package device

// Animation 定义了一个动画序列的行为
type Animation interface {
	// Run 执行动画的一个周期或直到被停止
	// executor: 用于执行姿态指令
	// stop: 接收停止信号的通道
	// speedMs: 动画执行的速度（毫秒）
	Run(executor PoseExecutor, stop <-chan struct{}, speedMs int) error
	// Name 返回动画的名称
	Name() string
}
