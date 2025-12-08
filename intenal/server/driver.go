package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"taxi-hailing/intenal/domain"
	"taxi-hailing/intenal/service"
	"taxi-hailing/pkg"

	"github.com/google/uuid"
)

type driverServer struct {
	srv http.Server
}

func NewDriverServer(port uint16, sec string, use *service.DriverService) *driverServer {
	mux := http.NewServeMux()
	hand := &driverHandler{[]byte(sec), use}
	// mux.HandleFunc("POST /drivers/register", hand.registerDriver)
	// mux.HandleFunc("POST /drivers/login", hand.loginDriver)
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
	use    *service.DriverService
}

func (h *driverHandler) driverOnline(w http.ResponseWriter, r *http.Request) {
	claim, ok := r.Context().Value(userCtxKey).(*pkg.MyClaims)
	if !ok {
		errorWrite(w, http.StatusInternalServerError, fmt.Errorf("context error"))
		return
	}
	id := r.PathValue("driver_id")
	if claim.UserID != id {
		errorWrite(w, http.StatusInternalServerError, fmt.Errorf("driver id != token's id"))
		return
	}
	loc := new(domain.Location)
	err := json.NewDecoder(r.Body).Decode(loc)
	if err != nil {
		errorWrite(w, http.StatusBadRequest, err)
		return
	}
	err = validateLocation(loc.Lat, loc.Lng)
	if err != nil {
		errorWrite(w, http.StatusBadRequest, err)
		return
	}
	uid, err := uuid.Parse(id)
	if err != nil {
		errorWrite(w, http.StatusBadRequest, err)
		return
	}

	sessionID, err := h.use.SetToOnline(r.Context(), uid, loc)
	if err != nil {
		errorWrite(w, http.StatusBadRequest, err)
		return
	}
	res := &domain.DriverOnlineResponse{
		Status:    "AVAILABLE",
		SessionID: sessionID.String(),
		Message:   "You are now online and ready to accept rides",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func (h *driverHandler) driverOffline(w http.ResponseWriter, r *http.Request) {
	claim, ok := r.Context().Value(userCtxKey).(*pkg.MyClaims)
	if !ok {
		errorWrite(w, http.StatusInternalServerError, fmt.Errorf("context error"))
		return
	}
	id := r.PathValue("driver_id")
	if claim.UserID != id {
		errorWrite(w, http.StatusInternalServerError, fmt.Errorf("driver id != token's id"))
		return
	}
	res, err := h.use.SetToOffline(r.Context(), id)
	if err != nil {
		errorWrite(w, http.StatusBadRequest, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func (h *driverHandler) driverLocationUpdate(w http.ResponseWriter, r *http.Request) {
	claim, ok := r.Context().Value(userCtxKey).(*pkg.MyClaims)
	if !ok {
		errorWrite(w, http.StatusInternalServerError, fmt.Errorf("context error"))
		return
	}
	id := r.PathValue("driver_id")
	if claim.UserID != id {
		errorWrite(w, http.StatusInternalServerError, fmt.Errorf("driver id != token's id"))
		return
	}
	loc := new(domain.LocationUpdate)
	err := json.NewDecoder(r.Body).Decode(loc)
	if err != nil {
		errorWrite(w, http.StatusBadRequest, err)
		return
	}
	err = validateUpdateLocation(loc)
	if err != nil {
		errorWrite(w, http.StatusBadRequest, err)
		return
	}
	res, err := h.use.UpdateDriverLocation(r.Context(), id, loc)
	if err != nil {
		errorWrite(w, http.StatusBadRequest, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func (h *driverHandler) driverEnRoute(w http.ResponseWriter, r *http.Request) {
	claim, ok := r.Context().Value(userCtxKey).(*pkg.MyClaims)
	if !ok {
		errorWrite(w, http.StatusInternalServerError, fmt.Errorf("context error"))
		return
	}
	id := r.PathValue("driver_id")
	if claim.UserID != id {
		errorWrite(w, http.StatusInternalServerError, fmt.Errorf("driver id != token's id"))
		return
	}

	req := new(domain.DriverLocationMessage)
	err := json.NewDecoder(r.Body).Decode(req)
	if err != nil {
		errorWrite(w, http.StatusBadRequest, err)
		return
	}

	res, err := h.use.EnRoute(r.Context(), id, req)
	if err != nil {
		errorWrite(w, http.StatusBadRequest, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func (h *driverHandler) driverStart(w http.ResponseWriter, r *http.Request) {
	claim, ok := r.Context().Value(userCtxKey).(*pkg.MyClaims)
	if !ok {
		errorWrite(w, http.StatusInternalServerError, fmt.Errorf("context error"))
		return
	}
	id := r.PathValue("driver_id")
	if claim.UserID != id {
		errorWrite(w, http.StatusInternalServerError, fmt.Errorf("driver id != token's id"))
		return
	}

	req := new(domain.DriverLocationMessage)
	err := json.NewDecoder(r.Body).Decode(req)
	if err != nil {
		errorWrite(w, http.StatusBadRequest, err)
		return
	}
	res, err := h.use.Start(r.Context(), id, req)
	if err != nil {
		errorWrite(w, http.StatusBadRequest, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func (h *driverHandler) driverComplete(w http.ResponseWriter, r *http.Request) {
	claim, ok := r.Context().Value(userCtxKey).(*pkg.MyClaims)
	if !ok {
		errorWrite(w, http.StatusInternalServerError, fmt.Errorf("context error"))
		return
	}
	id := r.PathValue("driver_id")
	if claim.UserID != id {
		errorWrite(w, http.StatusInternalServerError, fmt.Errorf("driver id != token's id"))
		return
	}
	req := new(domain.CompleteRideRequest)
	err := json.NewDecoder(r.Body).Decode(req)
	if err != nil {
		errorWrite(w, http.StatusBadRequest, err)
		return
	}
	res, err := h.use.Complete(r.Context(), id, req)
	if err != nil {
		errorWrite(w, http.StatusBadRequest, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

