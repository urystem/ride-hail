package service

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"taxi-hailing/intenal/broker"
	"taxi-hailing/intenal/domain"
	"taxi-hailing/intenal/repo"
)

const (
	// rate_per_km  = 100
	// rate_per_min = 100
	avgSpeed = 40.0 // км/ч
)

type RideService struct {
	slogger *slog.Logger
	db      *repo.RideRepo
	rabbit  *broker.RideBroker
}

func NewRideService(ctx context.Context, slogger *slog.Logger, db *repo.RideRepo, rabbit *broker.RideBroker) *RideService {
	service := &RideService{
		slogger: slogger,
		db:      db,
		rabbit:  rabbit,
	}
	go service.statusUpdater(ctx)
	return service
}

func (s *RideService) RegisterPassenger(ctx context.Context, user *domain.User) (string, error) {
	defer s.slogger.Info("new passenger registred", "action", "registration passenger")
	return s.db.RegisterPassenger(ctx, user)
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
			s.slogger.Error("canot get the body", "action", "get body", "error", err)
			continue
		}
		err = s.statusUpdate(ctx, status)
		if err != nil {
			s.slogger.Error("cannot update status to "+status.Status, "action", "update status", "error", err)
		}
	}
}

func (s *RideService) statusUpdate(ctx context.Context, status *domain.RideStatusUpdate) error {
	switch status.Status {
	case "MATCHED":
		return s.db.RideMatchedUpdate(ctx, status)
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
