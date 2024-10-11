package main

import (
	"fmt"
	"sync"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/net/websocket"
)

// クライアントの接続を管理するための変数
var clients = make(map[*websocket.Conn]bool)
var broadcast = make(chan string)
var mutex = &sync.Mutex{} // 同時アクセスを制御するためのミューテックス

func handleWebSocket(c echo.Context) error {
	websocket.Handler(func(ws *websocket.Conn) {
		// クライアントをリストに追加
		mutex.Lock()
		clients[ws] = true
		mutex.Unlock()

		defer func() {
			// クライアントが切断されたらリストから削除
			mutex.Lock()
			delete(clients, ws)
			mutex.Unlock()
			ws.Close()
		}()

		// 初回メッセージを送信
		err := websocket.Message.Send(ws, "Server: Hello, Client!")
		if err != nil {
			c.Logger().Error(err)
		}

		for {
			// クライアントからのメッセージを受信
			msg := ""
			err = websocket.Message.Receive(ws, &msg)
			if err != nil {
				c.Logger().Error(err)
				break // エラーが発生した場合、接続を閉じる
			}

			// 受け取ったメッセージをブロードキャストチャネルに送信
			broadcast <- fmt.Sprintf("Client says: \"%s\"", msg)
		}
	}).ServeHTTP(c.Response(), c.Request())
	return nil
}

func handleMessages() {
	for {
		// ブロードキャストチャネルからメッセージを受信
		msg := <-broadcast

		// 全クライアントにメッセージを送信
		mutex.Lock()
		for client := range clients {
			err := websocket.Message.Send(client, msg)
			if err != nil {
				fmt.Println("Error sending message:", err)
				client.Close()
				delete(clients, client)
			}
		}
		mutex.Unlock()
	}
}

func main() {
	e := echo.New()
	e.Use(middleware.Logger())
	e.Static("/", "public")

	// WebSocketハンドラー
	e.GET("/ws", handleWebSocket)

	// 別のゴルーチンでメッセージのハンドリングを行う
	go handleMessages()

	// サーバーを開始
	e.Logger.Fatal(e.Start(":8080"))
}
