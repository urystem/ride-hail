package ws

import (
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type authMessage struct {
	Type  string `json:"type"`
	Token string `json:"token"`
}

type myWebSocket struct {
	once   sync.Once
	done   chan struct{}
	sendCh chan any // канал для отправки сообщений
}

func (s *myWebSocket) safeClose() {
	s.once.Do(func() {
		close(s.done)
		time.Sleep(5 * time.Second)
		close(s.sendCh)
	})
}
func (s *myWebSocket) pushToChannel(zat any) {
	select {
	case <-s.done:
		return
	case s.sendCh <- zat:
		return
	}
}
