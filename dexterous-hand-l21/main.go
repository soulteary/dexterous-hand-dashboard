package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

// CAN消息结构
type CanMessage struct {
	Interface string `json:"interface"`
	ID        uint32 `json:"id"`
	Data      []byte `json:"data"`
}

// 前端请求结构
type ControlRequest struct {
	Command string      `json:"command"`
	Value   interface{} `json:"value"`
}

// 指令配置
type CommandConfig struct {
	Code    byte
	DataLen int
}

// 指令配置映射
var commandConfigs = map[string]CommandConfig{
	"大拇指旋转":    {0x01, 1},
	"横摆":       {0x02, 5},
	"手指根部":     {0x03, 5},
	"大拇指2关节":   {0x04, 1},
	"指尖":       {0x06, 5},
	"大拇指旋转速度":  {0x09, 1},
	"横摆速度":     {0x0A, 5},
	"手指根部速度":   {0x0B, 5},
	"大拇指2关节速度": {0x0C, 1},
	"指尖速度":     {0x16, 5},
}

// 发送到CAN服务
func sendToCanService(msg CanMessage) error {
	jsonData, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("JSON编码错误: %v", err)
	}

	resp, err := http.Post(
		"http://localhost:5260/api/can",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return fmt.Errorf("CAN服务请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("CAN服务错误: HTTP %d", resp.StatusCode)
	}

	return nil
}

// 处理控制请求
func handleControl(w http.ResponseWriter, r *http.Request) {
	var req ControlRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("错误请求格式: %v", err)
		http.Error(w, "无效请求格式", http.StatusBadRequest)
		return
	}

	config, ok := commandConfigs[req.Command]
	if !ok {
		log.Printf("未知指令: %s", req.Command)
		http.Error(w, "未知指令", http.StatusBadRequest)
		return
	}

	// 参数处理
	var values []int
	switch v := req.Value.(type) {
	case float64: // 单值
		values = []int{int(v)}
	case []interface{}: // 数组值
		for _, val := range v {
			if num, ok := val.(float64); ok {
				values = append(values, int(num))
			} else {
				log.Printf("非数字参数: %v", val)
				http.Error(w, "参数必须为数字", http.StatusBadRequest)
				return
			}
		}
	default:
		log.Printf("无效参数类型: %T", req.Value)
		http.Error(w, "无效参数类型", http.StatusBadRequest)
		return
	}

	// 验证参数长度
	if len(values) != config.DataLen {
		log.Printf("参数数量错误: %s 期望%d, 实际%d",
			req.Command, config.DataLen, len(values))
		http.Error(w, fmt.Sprintf("参数数量错误: 需要%d个", config.DataLen),
			http.StatusBadRequest)
		return
	}

	// 构建数据载荷
	data := []byte{config.Code}
	for _, val := range values {
		if val < 0 || val > 255 {
			log.Printf("参数超出范围: %d (0-255)", val)
			http.Error(w, "参数必须在0-255范围内", http.StatusBadRequest)
			return
		}
		data = append(data, byte(val))
	}
	canDevices := QueryNumberofCanDevices()
	//fmt.Println("canDevices: ", canDevices)

	for _, device := range canDevices {
		canMsg := CanMessage{
			Interface: device,
			ID:        0xff,
			Data:      data,
		}
		// 发送到CAN服务
		if err := sendToCanService(canMsg); err != nil {
			log.Printf("发送错误: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		log.Printf("发送指令: %s = %v", req.Command, values)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "指令发送成功"})
	}

}

// 查询CAN设备列表
func QueryNumberofCanDevices() []string {
	type ResponseData struct {
		Count      int      `json:"count"`
		Interfaces []string `json:"interfaces"`
	}

	type ApiResponse struct {
		Status string       `json:"status"`
		Data   ResponseData `json:"data"`
	}

	resp, err := http.Get("http://localhost:5260/api/setup/available")
	if err != nil {
		log.Printf("获取CAN设备列表失败: %v", err)
		return []string{}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("读取响应失败: %v", err)
		return []string{}
	}

	var apiResponse ApiResponse
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		log.Printf("解析JSON失败: %v", err)
		return []string{}
	}
	//fmt.Println(apiResponse.Data.Interfaces)
	return apiResponse.Data.Interfaces
}

func main() {
	// 设置日志

	// 注册路由
	http.HandleFunc("/control", handleControl)
	http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("服务运行中"))
	})

	// 静态文件服务
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	// 启动服务
	port := ":8087"
	log.Printf("启动服务 http://localhost%s", port)
	log.Fatal(http.ListenAndServe(port, nil))
	fmt.Println("服务启动在 http://localhost:8087")
}
