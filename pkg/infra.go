package pkg

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewDB(ctx context.Context, cfg *DatabaseCfg) (*pgxpool.Pool, error) {
	dsn := fmt.Sprintf(
		"postgresql://%s:%s@%s:%d/%s",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Database,
	)
	db, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}
	return db, db.Ping(ctx)
}

// func NewRabbitMQ(ctx context.Context, cfg *RabbitMQCfg) (any, error) {
// 	dsn := fmt.Sprintf(
// 		"amqp://%s:%s@%s:%d/",
// 		cfg.User,
// 		cfg.Password,
// 		cfg.Host,
// 		cfg.Port)
// 	myConn, err := amqp091.Dial(dsn)
// }
