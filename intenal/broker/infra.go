package broker

import (
	"fmt"
	"log/slog"
	"sync/atomic"
	"taxi-hailing/pkg"
	"time"

	"github.com/rabbitmq/amqp091-go"
)

type rabbit struct {
	logger    *slog.Logger
	conn      *amqp091.Connection
	connClose chan *amqp091.Error
	notifyCh  *amqp091.Channel
	isClosed  atomic.Bool
}

func NewRideRabbit(cfg pkg.RabbitMQCfg, slogger *slog.Logger) (any, error) {
	dsn := fmt.Sprintf("amqp://%s:%s@%s:%d/", cfg.User, cfg.Password, cfg.Host, cfg.Port)
	myRab := &rabbit{
		logger: slogger,
	}

	err := myRab.createChannel(dsn)
	if err != nil {
		return nil, err
	}

	go myRab.reconnectConn(dsn)
	return myRab, nil
}

func (r *rabbit) reconnectConn(url string) {
	for {
		<-r.connClose
		if r.isClosed.Load() {
			return
		}
		r.logger.Warn("rabbitMQ not working")
		for {
			if r.isClosed.Load() {
				return
			}
			r.logger.Info("trying to connect to rabbitmq")
			err := r.createChannel(url)
			if err != nil {
				time.Sleep(3 * time.Second)
				continue
			}
			r.logger.Info("connected to rabbitmq")
			break
		}
	}
}
