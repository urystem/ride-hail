package broker

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync/atomic"
	"taxi-hailing/intenal/domain"
	"taxi-hailing/pkg"
	"time"

	"github.com/rabbitmq/amqp091-go"
)

type DriverBroker struct {
	logger    *slog.Logger
	conn      *amqp091.Connection
	connClose chan *amqp091.Error
	ch        *amqp091.Channel //for publish
	req       chan *request
	isClosed  atomic.Bool
}

type request struct {
	req *amqp091.Delivery
}

func NewDriverRabbit(cfg pkg.RabbitMQCfg, slogger *slog.Logger) (*DriverBroker, error) {
	dsn := fmt.Sprintf("amqp://%s:%s@%s:%d/", cfg.User, cfg.Password, cfg.Host, cfg.Port)
	myRab := &DriverBroker{
		logger: slogger,
		req:    make(chan *request),
	}

	err := myRab.createChannel(dsn)
	if err != nil {
		return nil, err
	}

	go myRab.reconnectConn(dsn)
	return myRab, nil
}

func (d *DriverBroker) reconnectConn(dsn string) {
	for {
		<-d.connClose
		if d.isClosed.Load() {
			return
		}
		d.logger.Warn("rabbitMQ not working")
		for {
			if d.isClosed.Load() {
				return
			}
			d.logger.Info("trying to connect to rabbitmq")
			err := d.createChannel(dsn)
			if err != nil {
				time.Sleep(3 * time.Second)
				continue
			}
		}
	}
}

func (r *DriverBroker) createChannel(dsn string) error {
	myConn, err := amqp091.Dial(dsn)
	if err != nil {
		return err
	}
	r.conn = myConn
	r.connClose = make(chan *amqp091.Error)
	r.conn.NotifyClose(r.connClose)
	ch, err := r.conn.Channel()
	if err != nil {
		return errors.Join(r.conn.Close(), err)
	}
	r.ch = ch

	err = ch.ExchangeDeclare(
		"driver_topic", // имя exchange
		"topic",        // тип (direct, fanout, topic, headers)
		true,           // durable
		false,          // auto-deleted
		false,          // internal
		false,          // no-wait
		nil,            // args
	)
	if err != nil {
		return errors.Join(r.conn.Close(), err)
	}

	que, err := ch.QueueDeclare("driver_responses", true, false, false, false, nil)
	if err != nil {
		return errors.Join(r.conn.Close(), err)
	}
	err = ch.QueueBind(que.Name, "driver.response.*", "driver_topic", false, nil)
	if err != nil {
		return errors.Join(r.conn.Close(), err)
	}

	err = ch.ExchangeDeclare(
		"ride_topic", // имя exchange
		"topic",      // тип (direct, fanout, topic, headers)
		true,         // durable
		false,        // auto-deleted
		false,        // internal
		false,        // no-wait
		nil,          // args
	)
	if err != nil {
		return errors.Join(r.conn.Close(), err)
	}

	q1, err := ch.QueueDeclare("ride_requests", true, false, false, false, nil)
	if err != nil {
		return errors.Join(r.conn.Close(), err)
	}
	requests, err := ch.Consume(
		q1.Name,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return errors.Join(r.conn.Close(), err)
	}
	
	go func() {
		for req := range requests {
			r.req <- r.newRideRequest(&req)
		}
	}()

	return nil
}

func (r *DriverBroker) newRideRequest(req *amqp091.Delivery) *request {
	return &request{
		req: req,
	}
}

func (r *request) GiveBody() (*domain.RideRequestRabbit, error) {
	req := new(domain.RideRequestRabbit)
	err := json.Unmarshal(r.req.Body, req)
	if err != nil {
		return nil, err
	}
	return req, nil
}

func (r *DriverBroker) CloseRabbit() error {
	r.isClosed.Store(true)
	defer r.logger.Info("rabbit closed")
	return r.conn.Close()
}
