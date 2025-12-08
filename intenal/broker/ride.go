package broker

import (
	"context"
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

type RideBroker struct {
	logger         *slog.Logger
	conn           *amqp091.Connection
	connClose      chan *amqp091.Error
	status         chan *statusStu
	locationUpdate chan *locationStu
	drRespone      chan *matchResponse
	ch             *amqp091.Channel //for publish with ch
	isClosed       atomic.Bool
}

func NewRideRabbit(cfg pkg.RabbitMQCfg, slogger *slog.Logger) (*RideBroker, error) {
	dsn := fmt.Sprintf("amqp://%s:%s@%s:%d/", cfg.User, cfg.Password, cfg.Host, cfg.Port)
	myRab := &RideBroker{
		logger:         slogger,
		status:         make(chan *statusStu),
		locationUpdate: make(chan *locationStu),
	}

	err := myRab.createChannel(dsn)
	if err != nil {
		return nil, err
	}

	go myRab.reconnectConn(dsn)
	return myRab, nil
}

func (r *RideBroker) GiveStatusChannel() <-chan *statusStu {
	return r.status
}

func (r *RideBroker) CloseRabbit() error {
	r.isClosed.Store(true)
	defer r.logger.Info("rabbit closed")
	return r.conn.Close()
}

func (r *RideBroker) reconnectConn(url string) {
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

func (r *RideBroker) createChannel(dsn string) error {
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
	err = ch.QueueBind(q1.Name, "ride.request.*", "ride_topic", false, nil)
	if err != nil {
		return errors.Join(r.conn.Close(), err)
	}

	q2, err := ch.QueueDeclare("ride_status", true, false, false, false, nil)
	if err != nil {
		return errors.Join(r.conn.Close(), err)
	}
	// err = ch.QueueBind(q2.Name, "ride.status.*", "ride_topic", false, nil)
	// if err != nil {
	// 	return errors.Join(r.conn.Close(), err)
	// }

	msgs, err := ch.Consume(
		q2.Name,
		"",
		true, // manual ack
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return errors.Join(r.conn.Close(), err)
	}
	go func() {
		for msg := range msgs {
			r.status <- r.newStatus(&msg)
		}
	}()

	//location update
	err = ch.ExchangeDeclare(
		"location_fanout", // имя exchange
		"fanout",          // тип (direct, fanout, topic, headers)
		true,              // durable
		false,             // auto-deleted
		false,             // internal
		false,             // no-wait
		nil,               // args
	)
	if err != nil {
		return errors.Join(r.conn.Close(), err)
	}

	q3, err := ch.QueueDeclare(
		"",    // name
		false, // durable
		false, // delete when unused
		true,  // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return errors.Join(r.conn.Close(), err)
	}

	err = ch.QueueBind(
		q3.Name,           // queue name
		"",                // routing key
		"location_fanout", // exchange
		false,
		nil,
	)
	if err != nil {
		return errors.Join(r.conn.Close(), err)
	}

	msgs2, err := ch.Consume(
		q3.Name, // queue
		"",      // consumer
		true,    // auto-ack
		false,   // exclusive
		false,   // no-local
		false,   // no-wait
		nil,     // args
	)
	if err != nil {
		return errors.Join(r.conn.Close(), err)
	}

	go func() {
		for msg := range msgs2 {
			r.locationUpdate <- r.newLocation(&msg)
		}
	}()

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

	q4, err := ch.QueueDeclare("driver_responses", true, false, false, false, nil)
	if err != nil {
		return errors.Join(r.conn.Close(), err)
	}
	// err = ch.QueueBind(q4.Name, "driver.response.*", "driver_topic", false, nil)
	// if err != nil {
	// 	return errors.Join(r.conn.Close(), err)
	// }

	msgs3, err := ch.Consume(
		q4.Name,
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
		for mgs := range msgs3 {
			r.drRespone <- r.newDriverRespone(&mgs)
		}
	}()
	return nil
}

type statusStu struct {
	delivery *amqp091.Delivery
}

func (s *RideBroker) newStatus(hat *amqp091.Delivery) *statusStu {
	return &statusStu{
		hat,
	}
}

func (s *statusStu) GiveBody() (*domain.RideStatusUpdate, error) {
	status := new(domain.RideStatusUpdate)
	err := json.Unmarshal(s.delivery.Body, status)
	if err != nil {
		return nil, err
	}
	return status, nil
}

func (s *RideBroker) PublishRide(ctx context.Context, priority uint8, req *domain.RideRequestRabbit) error {
	b, err := json.Marshal(req)
	if err != nil {
		return err
	}

	return s.ch.PublishWithContext(
		ctx,
		"ride_topic",
		fmt.Sprintf("ride.request.%s", req.RideType),
		false,
		false,
		amqp091.Publishing{
			ContentType: "application/json",
			Body:        b,
			Priority:    priority,
		},
	)
}

type locationStu struct {
	location *amqp091.Delivery
}

func (s *RideBroker) newLocation(hat *amqp091.Delivery) *locationStu {
	return &locationStu{
		location: hat,
	}
}

func (l *locationStu) GiveBody() (*domain.DriverLocationUpdate, error) {
	loc := new(domain.DriverLocationUpdate)
	err := json.Unmarshal(l.location.Body, loc)
	if err != nil {
		return nil, err
	}
	return loc, nil
}

func (s *RideBroker) GiveLocationChannel() <-chan *locationStu {
	return s.locationUpdate
}

type matchResponse struct {
	delivery *amqp091.Delivery
}

func (s *RideBroker) newDriverRespone(hat *amqp091.Delivery) *matchResponse {
	return &matchResponse{
		delivery: hat,
	}
}

func (dr *matchResponse) GiveBody() (*domain.RideResponseMatch, error) {
	res := new(domain.RideResponseMatch)
	err := json.Unmarshal(dr.delivery.Body, res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (s *RideBroker) GiveResponeChannel() <-chan *matchResponse {
	return s.drRespone
}
