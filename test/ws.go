package main

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	// CheckOrigin: func(r *http.Request) bool {
	// 	return true // разрешить всем
	// },
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade error:", err)
		return
	}
	defer conn.Close()

	for {
		// читаем сообщение
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("read error:", err)
			return
		}

		log.Println("received:", string(msg))

		// отправляем назад
		err = conn.WriteMessage(websocket.TextMessage, []byte("echo: "+string(msg)))
		if err != nil {
			log.Println("write error:", err)
			return
		}
	}
}

func main() {
	http.HandleFunc("/ws", wsHandler)
	log.Println("server on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
