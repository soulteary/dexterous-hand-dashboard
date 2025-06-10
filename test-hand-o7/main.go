package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

type CanMessage struct {
	Interface string `json:"interface"`
	ID        uint32 `json:"id"`
	Data      []byte `json:"data"`
}

// sendToCanService 发送消息到CAN服务
func sendToCanService(msg CanMessage) error {
	jsonData, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("JSON编码错误: %v", err)
	}

	resp, err := http.Post("http://localhost:5260/api/can", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("CAN服务请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("CAN服务返回错误: HTTP %d", resp.StatusCode)
	}

	return nil
}

// handleControl 处理前端控制请求
func handleControl(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "只支持POST方法", http.StatusMethodNotAllowed)
		return
	}

	var msg CanMessage
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, fmt.Sprintf("请求解析失败: %v", err), http.StatusBadRequest)
		return
	}

	// 验证数据长度
	if len(msg.Data) < 1 {
		http.Error(w, "数据长度不足", http.StatusBadRequest)
		return
	}

	// 验证控制模式
	controlMode := msg.Data[0]
	if controlMode != 0x01 && controlMode != 0x05 {
		http.Error(w, "无效的控制模式", http.StatusBadRequest)
		return
	}

	// 发送到CAN服务
	if err := sendToCanService(msg); err != nil {
		http.Error(w, fmt.Sprintf("发送到CAN服务失败: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// serveStaticFile 提供静态文件服务
func serveStaticFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "只支持GET方法", http.StatusMethodNotAllowed)
		return
	}

	// 默认返回index.html
	http.ServeFile(w, r, "index.html")
}

func main() {
	// 设置路由
	http.HandleFunc("/", serveStaticFile)
	http.HandleFunc("/api/control", handleControl)

	// 启动服务器
	port := "8089"
	if envPort := os.Getenv("PORT"); envPort != "" {
		port = envPort
	}

	log.Printf("服务器启动，监听端口 %s...\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
