package define

// 配置结构体
type Config struct {
	CanServiceURL       string
	WebPort             string
	DefaultInterface    string
	AvailableInterfaces []string
}

// API 响应结构体
type ApiResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message,omitempty"`
	Error   string      `json:"error,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}
