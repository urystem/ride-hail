package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"taxi-hailing/intenal/domain"
)

type driverServer struct {
	srv http.Server
}

func NewDriverServer(port uint16, sec string, use any) *driverServer {
	mux := http.NewServeMux()
	hand := &driverHandler{[]byte(sec), use}
	mux.HandleFunc("POST /drivers/register", hand.registerDriver)
	mux.HandleFunc("POST /drivers/login", hand.loginDriver)
	// mux.HandleFunc("GET /drivers/info", hand.infoUser)
	mux.Handle("POST /drivers/{driver_id}/online", authMiddleware(http.HandlerFunc(hand.driverOnline), []byte(sec)))
	mux.Handle("POST /drivers/{driver_id}/offline", authMiddleware(http.HandlerFunc(hand.driverOffline), []byte(sec)))
	mux.Handle("POST /drivers/{driver_id}/location", authMiddleware(http.HandlerFunc(hand.driverLocationUpdate), []byte(sec)))
	mux.Handle("POST /drivers/{driver_id}/route", authMiddleware(http.HandlerFunc(hand.driverEnRoute), []byte(sec)))
	mux.Handle("POST /drivers/{driver_id}/start", authMiddleware(http.HandlerFunc(hand.driverStart), []byte(sec)))
	mux.Handle("POST /drivers/{driver_id}/complete", authMiddleware(http.HandlerFunc(hand.driverComplete), []byte(sec)))
	return &driverServer{
		srv: http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: mux,
		},
	}
}

func (s *driverServer) StartServer() error {
	return s.srv.ListenAndServe()
}

func (s *driverServer) ShutDownServer(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}

type driverHandler struct {
	secret []byte
	use    any
}

func (h *driverHandler) registerDriver(w http.ResponseWriter, r *http.Request) {
	user := new(domain.User)
	err := json.NewDecoder(r.Body).Decode(user)
	if err != nil {
		errorWrite(w, http.StatusBadRequest, err)
		return
	}
}

func (h *driverHandler) loginDriver(w http.ResponseWriter, r *http.Request) {

}

func (h *driverHandler) driverOnline(w http.ResponseWriter, r *http.Request) {
}

func (h *driverHandler) driverOffline(w http.ResponseWriter, r *http.Request) {
}
func (h *driverHandler) driverLocationUpdate(w http.ResponseWriter, r *http.Request) {
}

func (h *driverHandler) driverEnRoute(w http.ResponseWriter, r *http.Request) {
}

func (h *driverHandler) driverStart(w http.ResponseWriter, r *http.Request) {
}

func (h *driverHandler) driverComplete(w http.ResponseWriter, r *http.Request) {
}
