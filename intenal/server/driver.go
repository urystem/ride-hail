package server

import (
	"fmt"
	"net/http"
)

type driverServer struct {
	srv http.Server
}

func NewDriverServer(port uint16, sec string, use any) *rideServer {
	mux := http.NewServeMux()
	hand := &driverHandler{[]byte(sec), use}
	mux.HandleFunc("POST /drivers/register", hand.registerDriver)
	mux.HandleFunc("POST /drivers/login", hand.loginDriver)
	// mux.HandleFunc("GET /drivers/info", hand.infoUser)
	// mux.Handle("POST /rides", hand.authMiddleware(http.HandlerFunc(hand.createRide)))
	// mux.HandleFunc("POST /rides/{ride_id}/cancel", hand.cancelRide)
	return &rideServer{
		srv: http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: mux,
		},
	}
}

type driverHandler struct {
	secret []byte
	use    any
}

func (h *driverHandler) registerDriver(w http.ResponseWriter, r *http.Request) {

}

func (h *driverHandler) loginDriver(w http.ResponseWriter, r *http.Request) {

}
