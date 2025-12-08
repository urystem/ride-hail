package service

import (
	"context"
	"log/slog"
	"taxi-hailing/intenal/broker"
	"taxi-hailing/intenal/domain"
	"taxi-hailing/intenal/repo"
	"taxi-hailing/intenal/ws"
	"time"

	"github.com/google/uuid"
)

type DriverService struct {
	slogger *slog.Logger
	db      *repo.DriverRepo
	rabbit  *broker.DriverBroker
	ws      *ws.DriverHub
}

func NewDriverService(slogger *slog.Logger, db *repo.DriverRepo, rabbit *broker.DriverBroker, ws *ws.DriverHub) *DriverService {
	return &DriverService{
		slogger: slogger,
		db:      db,
		rabbit:  rabbit,
		ws:      ws,
	}
}

func (d *DriverService) SetToOnline(ctx context.Context, id uuid.UUID, loc *domain.Location) (uuid.UUID, error) {
	return d.db.UpdateDriverToOnline(ctx, id, loc)
}

func (d *DriverService) SetToOffline(ctx context.Context, id string) (*domain.DriverOfflineResponse, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}
	sessionID, err := d.db.UpdateDriverToOffline(ctx, uid)
	if err != nil {
		return nil, err
	}

	summary, err := d.db.GetDriverSessionSummary(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	return &domain.DriverOfflineResponse{
		Status:         "OFFLINE",
		SessionID:      uid.String(),
		SessionSummary: *summary,
		Message:        "You are now offline",
	}, nil
}

func (d *DriverService) UpdateDriverLocation(ctx context.Context, id string, loc *domain.LocationUpdate) (*domain.DriverCoordinateUpdate, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}

	hisID, err := d.db.UpdateDriverLocation(ctx, uid, loc)
	if err != nil {
		return nil, err
	}

	return &domain.DriverCoordinateUpdate{
		CoordinateID: hisID.String(),
		UpdatedAt:    time.Now().UTC().Format("2006-01-02T15:04:05Z"),
	}, nil
}

func (d *DriverService) EnRoute(ctx context.Context, id string, req *domain.DriverLocationMessage) (*domain.DriverStartRideResponse, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}

	err = d.db.UpdateDriverToEnRoute(ctx, uid, req)
	if err != nil {
		return nil, err
	}
	return &domain.DriverStartRideResponse{
		RideID:    req.RideID,
		Status:    "EN_ROUTE",
		StartedAt: time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		Message:   "Ride enroute successfully",
	}, nil
}

func (d *DriverService) Start(ctx context.Context, id string, req *domain.DriverLocationMessage) (*domain.DriverStartRideResponse, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}
	err = d.db.UpdateDriverToBusy(ctx, uid, req)
	if err != nil {
		return nil, err
	}
	return &domain.DriverStartRideResponse{
		RideID:    req.RideID,
		Status:    "BUSY",
		StartedAt: time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		Message:   "Ride started successfully",
	}, nil
}

func (d *DriverService) Complete(ctx context.Context, id string, req *domain.CompleteRideRequest) (*domain.DriverCompleteRideResponse, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}
	fare, err := d.db.CompleteRide(ctx, uid, req)
	if err != nil {
		return nil, err
	}
	return &domain.DriverCompleteRideResponse{
		RideID:         req.RideID,
		Status:         "AVAILABLE",
		CompletedAt:    time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		DriverEarnings: fare,
		Message:        "Ride completed successfully",
	}, nil
}

// func (d *DriverService) 