package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"taxi-hailing/intenal/domain"
	"taxi-hailing/intenal/service"
	"taxi-hailing/pkg"
)

type rideServer struct {
	srv http.Server
}

func NewRideServer(port uint16, sec string, use *service.RideService) *rideServer {
	mux := http.NewServeMux()
	hand := &rideHandler{[]byte(sec), use}
	mux.HandleFunc("POST /register", hand.registerPassenger)
	mux.HandleFunc("POST /login", hand.loginPassenger)
	mux.HandleFunc("GET /user/info", hand.infoUser)
	mux.Handle("POST /rides", authMiddleware(http.HandlerFunc(hand.createRide), []byte(sec)))
	mux.HandleFunc("POST /rides/{ride_id}/cancel", hand.cancelRide)
	return &rideServer{
		srv: http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: mux,
		},
	}
}

func (s *rideServer) StartServer() error {
	return s.srv.ListenAndServe()
}

func (s *rideServer) ShutDownServer(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}

type rideHandler struct {
	secret []byte
	use    *service.RideService
}

func (h *rideHandler) registerPassenger(w http.ResponseWriter, r *http.Request) {
	user := new(domain.User)
	err := json.NewDecoder(r.Body).Decode(user)
	if err != nil {
		errorWrite(w, http.StatusBadRequest, err)
		return
	}
	err = validateUserInput(user, true)
	if err != nil {
		errorWrite(w, http.StatusBadRequest, err)
		return
	}
	user.PasswordHash, err = pkg.HashPassword(user.PasswordHash, h.secret)
	if err != nil {
		errorWrite(w, http.StatusInternalServerError, err)
		return
	}
	id, err := h.use.RegisterPassenger(r.Context(), user)
	if err != nil {
		errorWrite(w, http.StatusInternalServerError, err)
		return
	}

	claims := &pkg.MyClaims{
		UserID: id,
		Name:   user.Name,
		Email:  user.Email,
		Role:   user.Role,
	}

	token, err := pkg.GenerateTokenMyClaims(claims, h.secret)
	if err != nil {
		errorWrite(w, http.StatusInternalServerError, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pkg.RegistrationResponse{ID: id, Token: token})
}

func (h *rideHandler) loginPassenger(w http.ResponseWriter, r *http.Request) {
	user := new(domain.User)
	err := json.NewDecoder(r.Body).Decode(user)
	if err != nil {
		errorWrite(w, http.StatusBadRequest, err)
		return
	}
	err = validateUserInput(user, false)
	if err != nil {
		errorWrite(w, http.StatusBadRequest, err)
		return
	}

	ourUser, err := h.use.GetUserByEmail(r.Context(), user.Email)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			errorWrite(w, http.StatusNotFound, err)
		} else {
			errorWrite(w, http.StatusBadRequest, err)
		}
		return
	}
	check, err := pkg.CheckPassword(user.PasswordHash, ourUser.PasswordHash, h.secret)
	if err != nil {
		errorWrite(w, http.StatusInternalServerError, err)
		return
	}
	if !check {
		errorWrite(w, http.StatusBadRequest, fmt.Errorf("wrong password"))
		return
	}

	if ourUser.Status == "BANNED" {
		errorWrite(w, http.StatusBadRequest, fmt.Errorf("wrong status: %s", ourUser.Status))
		return
	}

	claims := &pkg.MyClaims{
		UserID: ourUser.ID,
		Name:   ourUser.Name,
		Email:  user.Email,
		Role:   ourUser.Role,
	}

	token, err := pkg.GenerateTokenMyClaims(claims, h.secret)
	if err != nil {
		errorWrite(w, http.StatusInternalServerError, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pkg.RegistrationResponse{ID: ourUser.ID, Token: token})
}

func (h *rideHandler) infoUser(w http.ResponseWriter, r *http.Request) {
	claim, err := getClaim(r, h.secret)
	if err != nil {
		errorWrite(w, http.StatusBadRequest, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(claim)
}

func (h *rideHandler) createRide(w http.ResponseWriter, r *http.Request) {
	claim, ok := r.Context().Value(userCtxKey).(*pkg.MyClaims)
	if !ok {
		errorWrite(w, http.StatusInternalServerError, fmt.Errorf("context error"))
		return
	}

	ride := new(domain.RideRequest)
	err := json.NewDecoder(r.Body).Decode(ride)
	if err != nil {
		errorWrite(w, http.StatusBadRequest, err)
		return
	}
	err = validatorRide(ride)
	if err != nil {
		errorWrite(w, http.StatusBadRequest, err)
		return
	}
	if ride.PassengerID != claim.UserID {
		errorWrite(w, http.StatusBadRequest, fmt.Errorf("wrong id"))
		return
	}

	res, err := h.use.CreateRide(r.Context(), ride)
	if err != nil {
		errorWrite(w, http.StatusBadRequest, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)

}

func (h *rideHandler) cancelRide(w http.ResponseWriter, r *http.Request) {

}

func validatorRide(ride *domain.RideRequest) error {
	if ride.PassengerID == "" {
		return fmt.Errorf("passenger_id is required")
	}

	if ride.PickupAddress == "" {
		return fmt.Errorf("pickup_address is required")
	}
	if ride.DestinationAddress == "" {
		return fmt.Errorf("destination_address is required")
	}

	if ride.PickupLatitude < -90 || ride.PickupLatitude > 90 {
		return fmt.Errorf("pickup_latitude must be between -90 and 90")
	}
	if ride.PickupLongitude < -180 || ride.PickupLongitude > 180 {
		return fmt.Errorf("pickup_longitude must be between -180 and 180")
	}
	if ride.DestinationLatitude < -90 || ride.DestinationLatitude > 90 {
		return fmt.Errorf("destination_latitude must be between -90 and 90")
	}
	if ride.DestinationLongitude < -180 || ride.DestinationLongitude > 180 {
		return fmt.Errorf("destination_longitude must be between -180 and 180")
	}

	if ride.RideType == "" {
		return fmt.Errorf("ride_type is required")
	}
	return nil
}
