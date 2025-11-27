package server

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"taxi-hailing/intenal/domain"
	"taxi-hailing/intenal/service"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type rideServer struct {
	srv http.Server
}

func NewRideServer(port uint16, sec string, use *service.RideService) *rideServer {
	mux := http.NewServeMux()
	hand := &rideHandler{[]byte(sec), use}
	mux.HandleFunc("POST /passenger/register", hand.registerPassenger)
	mux.HandleFunc("POST /passenger/login", hand.loginPassenger)
	mux.HandleFunc("POST /rides", hand.createRide)
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

type registrationResponse struct {
	Token string `json:"token"`
}

type myClaims struct {
	PassengerID string
	Name        string
	Email       string
	Role        string
	jwt.RegisteredClaims
}

func (h *rideHandler) generateTokenMyClaims(claims *myClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(h.secret))
}

func (h *rideHandler) parseTokenMyClaims(tokenStr string) (*myClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &myClaims{}, func(t *jwt.Token) (any, error) {
		return h.secret, nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*myClaims)
	if !ok {
		return nil, fmt.Errorf("invalid struture")
	}
	return claims, nil
}
func (h *rideHandler) hashPassword(password string) (string, error) {
	mac := hmac.New(sha256.New, h.secret)
	_, err := mac.Write([]byte(password))
	if err != nil {
		return "", err
	}
	//sum []byte қайтарады, әрі prefix қояды, бірақ парольға префикс керек емес
	return hex.EncodeToString(mac.Sum(nil)), nil
}

func (h *rideHandler) registerPassenger(w http.ResponseWriter, r *http.Request) {
	user := new(domain.User)
	err := json.NewDecoder(r.Body).Decode(user)
	if err != nil {
		errorWrite(w, http.StatusBadRequest, err)
		return
	}
	err = validateUserInput(user)
	if err != nil {
		errorWrite(w, http.StatusBadRequest, err)
		return
	}
	user.PasswordHash, err = h.hashPassword(user.PasswordHash)
	if err != nil {
		errorWrite(w, http.StatusInternalServerError, err)
		return
	}

	id, err := h.use.RegisterPassenger(r.Context(), user)
	if err != nil {
		errorWrite(w, http.StatusInternalServerError, err)
		return
	}
	
	claims := &myClaims{
		PassengerID: id,
		Name:        user.Name,
		Email:       user.Email,
		Role:        "PASSENGER",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token, err := h.generateTokenMyClaims(claims)
	if err != nil {
		errorWrite(w, http.StatusInternalServerError, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(registrationResponse{token})
}

func (h *rideHandler) loginPassenger(w http.ResponseWriter, r *http.Request) {
	user := new(domain.User)
	err := json.NewDecoder(r.Body).Decode(user)
	if err != nil {
		errorWrite(w, http.StatusBadRequest, err)
		return
	}
	err = validateUserInput(user)
	if err != nil {
		errorWrite(w, http.StatusBadRequest, err)
		return
	}

}

// func (h *rideHandler) authMiddleware(next http.Handler) http.Handler {
// 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

// 		auth := r.Header.Get("Authorization")
// 		if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
// 			http.Error(w, "missing bearer token", http.StatusUnauthorized)
// 			return
// 		}

// 		token := strings.TrimPrefix(auth, "Bearer ")

// 		userID, err := YourVerifyFunc(token)
// 		if err != nil {
// 			http.Error(w, "invalid token", http.StatusUnauthorized)
// 			return
// 		}

//			// кладем userID в контекст
//			ctx := context.WithValue(r.Context(), UserIDKey, userID)
//			next.ServeHTTP(w, r.WithContext(ctx))
//		})
//	}
func (h *rideHandler) createRide(w http.ResponseWriter, r *http.Request) {
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

}

func (h *rideHandler) cancelRide(w http.ResponseWriter, r *http.Request) {

}

type myErr struct {
	ErrStr string `json:"error"`
}

func errorWrite(w http.ResponseWriter, code int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	msg := &myErr{
		ErrStr: err.Error(),
	}
	json.NewEncoder(w).Encode(msg)
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

// ValidateUserInput валидирует name, email и пароль
func validateUserInput(user *domain.User) error {
	if len(strings.TrimSpace(user.Name)) == 0 {
		return errors.New("name cannot be empty")
	}
	if len(user.Name) < 2 {
		return errors.New("name too short, minimum 2 characters")
	}

	if len(strings.TrimSpace(user.Email)) == 0 {
		return errors.New("email cannot be empty")
	}
	// простая проверка email
	if !strings.Contains(user.Email, "@") || !strings.Contains(user.Email, ".") {
		return errors.New("invalid email format")
	}

	if len(user.PasswordHash) < 6 {
		return errors.New("password too short, minimum 6 characters")
	}

	return nil
}
