package service

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"taxi-hailing/intenal/broker"
	"taxi-hailing/intenal/domain"
	"taxi-hailing/intenal/repo"
	"taxi-hailing/intenal/ws"
	"time"
)

const (
	avgSpeed = 40.0 // км/ч
)

type RideService struct {
	slogger *slog.Logger
	db      *repo.RideRepo
	rabbit  *broker.RideBroker
	ws      *ws.PassengerHub
}

func NewRideService(ctx context.Context, slogger *slog.Logger, db *repo.RideRepo, rabbit *broker.RideBroker, ws *ws.PassengerHub) *RideService {
	service := &RideService{
		slogger: slogger,
		db:      db,
		rabbit:  rabbit,
		ws:      ws,
	}
	go service.statusUpdater(ctx)
	go service.rideMatcherService(ctx)
	go service.locationUpdater(ctx)
	return service
}

func (s *RideService) RegisterPassenger(ctx context.Context, user *domain.User) (string, error) {
	defer s.slogger.Info("new passenger registred", "action", "registration passenger")
	return s.db.RegisterPassenger(ctx, user)
}

func (s *RideService) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	return s.db.GetUserByEmail(ctx, email)
}

func (s *RideService) CreateRide(ctx context.Context, ride *domain.RideRequest) (*domain.RideResponse, error) {
	distance_km := distanceKM(ride.PickupLatitude, ride.PickupLongitude, ride.DestinationLatitude, ride.DestinationLongitude)
	duration_min := distance_km / avgSpeed * 60
	base_fare, rate_per_km, rate_per_min, priority := giveTypesFare(ride.RideType)
	fare := base_fare + (distance_km * rate_per_km) + (duration_min * rate_per_min)

	res := &domain.RideResponse{
		EstimatedDistanceKM:      distance_km,
		EstimatedDurationMinutes: int(duration_min),
		EstimatedFare:            fare,
		BaseFare:                 base_fare,
	}

	err := s.db.CreateRideTx(ctx, ride, res)
	if err != nil {
		return nil, err
	}

	req := &domain.RideRequestRabbit{
		RideID:     res.RideID,
		RideNumber: res.RideNumber,
		PickupLocation: domain.Coordinates{
			Lat:     ride.PickupLatitude,
			Lng:     ride.PickupLongitude,
			Address: ride.PickupAddress,
		},
		DestinationLocation: domain.Coordinates{
			Lat:     ride.DestinationLatitude,
			Lng:     ride.DestinationLongitude,
			Address: ride.DestinationAddress,
		},
		RideType:       ride.RideType,
		EstimatedFare:  fare,
		MaxDistanceKM:  distance_km,
		TimeoutSeconds: 2,
		CorrelationID:  "dd",
	}
	s.rabbit.PublishRide(ctx, priority, req)
	return res, nil
}

func (s *RideService) statusUpdater(ctx context.Context) {
	for v := range s.rabbit.GiveStatusChannel() {
		status, err := v.GiveBody()
		if err != nil {
			s.slogger.Error("canot get the body of status ride", "action", "get body", "error", err)
			continue
		}
		err = s.statusUpdate(ctx, status)
		if err != nil {
			s.slogger.Error("cannot update status to "+status.Status, "action", "update status", "error", err)
		}
	}
}

func (s *RideService) statusUpdate(ctx context.Context, status *domain.RideStatusUpdate) error {
	passengerID, err := s.db.GetPassengerIDByRideID(ctx, status.RideID)
	if err != nil {
		s.slogger.Error("cannnot get passenger id", "error", err)
		return err
	}
	answerWS := &domain.RideStatusUpdate{
		RideID:        status.RideID,
		Status:        status.Status,
		Timestamp:     time.Now(),
		DriverID:      status.DriverID,
		CorrelationID: status.CorrelationID,
	}
	defer s.ws.GiveToPassenger(passengerID, answerWS)
	switch status.Status {
	// case "MATCHED":
	// 	return s.db.RideMatchedUpdate(ctx, status)
	// 	// s.ws.GiveToPassenger(sta)
	case "EN_ROUTE":
		return s.db.RideEnRouteUpdate(ctx, status)
	case "ARRIVED":
		return s.db.RideArrivedUpdate(ctx, status)
	case "IN_PROGRESS":
		return s.db.RideInProgressUpdate(ctx, status)
	case "COMPLETED":
		return s.db.RideCompleteUpdate(ctx, status)
	default:
		return fmt.Errorf("invalid status: %s", status.Status)
	}
}

func (s *RideService) rideMatcherService(ctx context.Context) {
	for v := range s.rabbit.GiveResponeChannel() {
		res, err := v.GiveBody()
		if err != nil {
			s.slogger.Error("canot get the body of respone driver", "action", "get body", "error", err)
			continue
		}
		err = s.rideMatched(ctx, res)
		if err != nil {
			s.slogger.Error("cannot update to match status", "action", "update status", "error", err)
		}
	}
}

func (s *RideService) rideMatched(ctx context.Context, match *domain.RideResponseMatch) error {
	passengerID, err := s.db.GetPassengerIDByRideID(ctx, match.RideID)
	if err != nil {
		s.slogger.Error("cannnot get passenger id", "error", err)
		return err
	}
	rideNum, err := s.db.GetRideNumberByRideID(ctx, match.RideID)
	if err != nil {
		s.slogger.Error("cannnot get passenger id", "error", err)
		return err
	}

	err = s.db.RideMatchedUpdate(ctx, match)
	if err != nil {
		return err
	}

	wsMatch := &domain.RideStatusUpdateMatched{
		Type:       "ride_status_update",
		RideID:     match.RideID,
		RideNumber: rideNum,
		Status:     "MATCHED",
		DriverInfo: domain.DriverInfoWs{
			DriverID:   match.DriverID,
			DriverInfo: match.DriverInfo,
		},
		CorrelationID: match.CorrelationID,
	}
	s.ws.GiveToPassenger(passengerID, wsMatch)
	return nil
}

func distanceKM(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371.0 // Радиус Земли в км

	// Перевод в радианы
	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180

	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Sin(dLon/2)*math.Sin(dLon/2)*math.Cos(lat1Rad)*math.Cos(lat2Rad)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c // расстояние в км
}

func giveTypesFare(str string) (float64, float64, float64, uint8) {
	switch str {
	case "PREMIUM":
		return 800, 120, 60, 5
	case "XL":
		return 1000, 150, 75, 10
	default: /*case "ECONOMY":*/
		return 500, 100, 50, 1
	}
}

func (s *RideService) CancelRide(ctx context.Context, rideID string, req *domain.CancelRideRequest) (*domain.CancelRideResponse, error) {
	err := s.db.CancelRide(ctx, rideID, req)
	if err != nil {
		return nil, err
	}

	return &domain.CancelRideResponse{
		RideID:      rideID,
		Status:      "Cancelled",
		CancelledAt: time.Now(),
		Message:     "Ride cancelled successfully",
	}, nil
}

func (s *RideService) locationUpdater(ctx context.Context) {
	for v := range s.rabbit.GiveLocationChannel() {
		loca, err := v.GiveBody()
		if err != nil {
			s.slogger.Error("canot get location update in body", "action", "get body", "error", err)
			continue
		}
		err = s.locationUpdateHelp(ctx, loca)
		if err != nil {
			s.slogger.Error("cannot update location", "error", err)
		}
	}
}

func (s *RideService) locationUpdateHelp(ctx context.Context, loca *domain.DriverLocationUpdate) error {
	myType, err := s.db.GetRideVehicleType(ctx, loca.RideID)
	if err != nil {
		return err
	}
	_, rate_per_km, rate_per_min, _ := giveTypesFare(myType)
	passID, err := s.db.GetPassengerIDByRideID(ctx, loca.RideID)
	if err != nil {
		return err
	}
	oldCoor, err := s.db.GetCurrentCoordinate(ctx, passID)
	if err != nil {
		return err
	}

	distanceKM := distanceKM(oldCoor.Latitude, oldCoor.Longitude, loca.Location.Lat, loca.Location.Lng)
	durationMIN := time.Since(oldCoor.UpdatedAt).Minutes()

	fare := (distanceKM * rate_per_km) + (durationMIN * rate_per_min)
	data := &domain.LocationCoordinateUpdate{
		DriverID:       loca.DriverID,
		RideID:         loca.RideID,
		Location:       loca.Location,
		FareAmount:     fare,
		Distance:       oldCoor.DistanceKm + distanceKM,
		DurationMinute: oldCoor.DurationMinutes + int(durationMIN),
	}

	err = s.db.RideLocationUpdate(ctx, data)
	if err != nil {
		return err
	}
	go s.ws.GiveToPassenger(passID, loca)
	return nil
}
