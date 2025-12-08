package ws

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"taxi-hailing/pkg"
	"time"

	"github.com/gorilla/websocket"
)

type PassengerHub struct {
	secret  []byte
	srv     *http.Server
	slogger *slog.Logger
	clients sync.Map // map[string]*MyWebSocket map[string] chan<- []byte
	// db      *repo.RideRepo
}

func (hub *PassengerHub) GiveToPassenger(id string, zat any) {
	wsStu, ok := hub.clients.Load(id)
	if !ok {
		return
	}
	ws, ok := wsStu.(*myWebSocket)
	if !ok {
		hub.slogger.Info("cannot parse myWebSocket")
		return
	}
	ws.pushToChannel(zat)
}

func (hub *PassengerHub) connectPassenger(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		hub.slogger.Error("upgrade error:", "error", err)
		return
	}
	defer conn.Close()
	id := r.PathValue("passenger_id")

	//Client sends authentication message within 5 seconds
	err = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	if err != nil {
		return
	}
	auth := new(authMessage)
	err = conn.ReadJSON(auth)
	if err != nil {
		hub.slogger.Error("websocket_auth_timeout", "error", err)
		conn.WriteJSON(map[string]string{"error": err.Error()})
		return
	}

	if auth.Type != "auth" {
		conn.WriteJSON(map[string]string{"error": fmt.Sprintf("invalid auth type: %s", auth.Type)})
		return
	}

	claim, err := pkg.ParseTokenMyClaims(auth.Token, hub.secret)
	if err != nil {
		conn.WriteJSON(map[string]string{"error": err.Error()})
		return
	}

	if claim.UserID != id {
		conn.WriteJSON(map[string]string{"error": fmt.Sprintln("wrong id != cliam id")})
		return
	}

	if claim.Role != "PASSENGER" {
		conn.WriteJSON(map[string]string{"error": fmt.Sprintln("wrong role != role")})
	}

	// user, err := hub.db.GetPassengerWS(r.Context(), id)
	// if err != nil {
	// 	conn.WriteJSON(map[string]string{"error": err.Error()})
	// 	return
	// }
	// if user.Role != "PASSENGER" {
	// 	conn.WriteJSON(map[string]string{"error": "your role is not passenger"})
	// 	return
	// }
	// if user.Status != "ACTIVE" {
	// 	conn.WriteJSON(map[string]string{"error": "your status is not active"})
	// 	return
	// }

	conn.WriteJSON(map[string]string{"msg": "please wait"})
	_, ok := hub.clients.Load(id)
	if ok {
		conn.WriteJSON(map[string]string{"error": "already connected in other ws"})
		return
	}

	myWS := &myWebSocket{
		done:   make(chan struct{}),
		sendCh: make(chan any),
	}
	go hub.pingPong(r.Context(), conn, myWS)

	hub.clients.Store(id, myWS)
	defer hub.clients.Delete(id)
	go hub.writer(conn, myWS)
	<-myWS.done
}

func (hub *PassengerHub) pingPong(ctx context.Context, ws *websocket.Conn, my *myWebSocket) {
	defer my.safeClose()
	const (
		pingPeriod = 30 * time.Second
		pongWait   = 60 * time.Second
	)

	// === 1. PONG handler ===
	ws.SetReadDeadline(time.Now().Add(pongWait))
	ws.SetPongHandler(func(_ string) error {
		// каждый pong от клиента — обновляем таймер
		ws.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	ws.PingHandler()
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := ws.WriteControl(websocket.PingMessage, []byte("ping"), time.Now().Add(5*time.Second)); err != nil {
				ws.Close()
				return
			}
		}
	}
}

func (hub *PassengerHub) writer(conn *websocket.Conn, ws *myWebSocket) {
	defer ws.safeClose()
	for data := range ws.sendCh {
		err := conn.WriteJSON(data)
		if err != nil {
			return
		}
	}
}

func NewWebSocket(slogger *slog.Logger, secret []byte, port uint16) *PassengerHub {
	mux := http.NewServeMux()
	my := &PassengerHub{
		secret:  secret,
		slogger: slogger,
		// db:      db,
	}
	mux.HandleFunc("/ws/passengers/{passenger_id}", my.connectPassenger)
	// mux.HandleFunc("GET /ws", wsHandler)
	return &PassengerHub{
		srv: &http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: mux,
		},
	}
}

func (hub *PassengerHub) StartServer() error {
	return hub.srv.ListenAndServe()
}

func (hub *PassengerHub) CloseServer() error {
	defer hub.clients.Clear()
	return hub.srv.Close()
}

// func (w *webSocketPassengers)
