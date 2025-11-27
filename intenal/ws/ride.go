package ws

import (
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"taxi-hailing/intenal/repo"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type myWebSocket struct {
	done   chan struct{}
	SendCh chan []byte // канал для отправки сообщений
}

type passengerHub struct {
	slogger slog.Logger
	clients sync.Map // map[string]*MyWebSocket map[string] chan<- []byte
	db      repo.RideRepo
}

func (hub *passengerHub) connectPassenger(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		hub.slogger.Error("upgrade error:", "error", err)
		return
	}
	defer conn.Close()
	id := r.PathValue("passenger_id")
	user, err := hub.db.GetPassengerWS(r.Context(), id)

}

func NewWebSocket(port uint16, use any) {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws/passengers/{passenger_id}", nil)
	// mux.HandleFunc("GET /ws", wsHandler)
	server := http.Server{
		Addr:    fmt.Sprintf(":%d", 8080),
		Handler: mux,
	}
	server.ListenAndServe()
}

// func (w *webSocketPassengers)
