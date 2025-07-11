package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许跨域
	},
}

// 场景1: 服务器接收消息后突然断开连接（不发送关闭帧）
func scenario1Handler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("升级错误:", err)
		return
	}
	defer conn.Close()

	log.Println("场景1: 客户端已连接 - 将在收到第一条消息后异常断开")

	// 读取一条消息
	_, message, err := conn.ReadMessage()
	if err != nil {
		log.Println("读取错误:", err)
		return
	}
	log.Printf("收到消息: %s", message)

	// 模拟异常：直接关闭底层连接，不发送WebSocket关闭帧
	conn.UnderlyingConn().Close()
}

// 场景2: 服务器在心跳期间停止响应
func scenario2Handler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("升级错误:", err)
		return
	}
	defer conn.Close()

	log.Println("场景2: 客户端已连接 - 将在10秒后停止响应")

	// 设置读取超时
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))

	// 持续读取消息直到超时
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("读取超时或错误:", err)
			// 不发送关闭帧，直接返回
			return
		}
		log.Printf("收到消息: %s", message)
	}
}

// 场景3: 服务器在随机时间后崩溃
func scenario3Handler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("升级错误:", err)
		return
	}
	defer conn.Close()

	log.Println("场景3: 客户端已连接 - 将在5-15秒内随机崩溃")

	// 启动一个goroutine模拟随机崩溃
	go func() {
		time.Sleep(time.Duration(5+time.Now().Unix()%10) * time.Second)
		log.Println("模拟服务器崩溃！")
		panic("模拟服务器崩溃")
	}()

	// 正常处理消息
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("读取错误:", err)
			return
		}
		log.Printf("收到消息: %s", message)

		// 回显消息
		err = conn.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			log.Println("写入错误:", err)
			return
		}
	}
}

// 场景4: 代理超时模拟（长时间无数据传输）
func scenario4Handler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("升级错误:", err)
		return
	}
	defer conn.Close()

	log.Println("场景4: 客户端已连接 - 模拟代理超时（60秒无数据）")

	// 设置较长的读取截止时间
	conn.SetReadDeadline(time.Now().Add(70 * time.Second))

	// 读取第一条消息
	_, message, err := conn.ReadMessage()
	if err != nil {
		log.Println("读取错误:", err)
		return
	}
	log.Printf("收到第一条消息: %s，之后将保持静默", message)

	// 保持连接但不发送任何数据，模拟代理超时
	time.Sleep(65 * time.Second)
}

// 正常的WebSocket处理器（用于对比）
func normalHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("升级错误:", err)
		return
	}
	defer conn.Close()

	log.Println("正常处理器: 客户端已连接")

	// 设置心跳响应
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// 启动心跳检测
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					return
				}
			}
		}
	}()

	// 正常处理消息
	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("读取错误:", err)
			return
		}
		log.Printf("收到消息: %s", message)

		// 回显消息
		err = conn.WriteMessage(messageType, message)
		if err != nil {
			log.Println("写入错误:", err)
			return
		}
	}
}

func main() {
	// 注册不同场景的处理器
	http.HandleFunc("/ws/scenario1", scenario1Handler) // 异常断开
	http.HandleFunc("/ws/scenario2", scenario2Handler) // 读取超时
	http.HandleFunc("/ws/scenario3", scenario3Handler) // 服务器崩溃
	http.HandleFunc("/ws/scenario4", scenario4Handler) // 代理超时
	http.HandleFunc("/ws/normal", normalHandler)       // 正常处理

	// 提供静态文件服务
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})

	fmt.Println("WebSocket服务器启动在 :8080")
	fmt.Println("访问 http://localhost:8080 查看测试页面")
	fmt.Println("\n可用的WebSocket端点:")
	fmt.Println("- ws://localhost:8080/ws/scenario1 - 收到消息后异常断开")
	fmt.Println("- ws://localhost:8080/ws/scenario2 - 10秒后停止响应")
	fmt.Println("- ws://localhost:8080/ws/scenario3 - 随机时间崩溃")
	fmt.Println("- ws://localhost:8080/ws/scenario4 - 模拟代理超时")
	fmt.Println("- ws://localhost:8080/ws/normal - 正常工作")

	log.Fatal(http.ListenAndServe(":8080", nil))
}
