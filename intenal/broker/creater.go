package broker

import (
	"errors"

	"github.com/rabbitmq/amqp091-go"
)

func (r *rabbit) createChannel(dsn string) error {
	myConn, err := amqp091.Dial(dsn)
	if err != nil {
		return err
	}
	r.conn = myConn
	r.connClose = make(chan *amqp091.Error)
	r.conn.NotifyClose(r.connClose)
	orderCh, err := r.conn.Channel()
	if err != nil {
		return errors.Join(r.conn.Close(), err)
	}
	_=orderCh
	// err = r.declareOrderTopic(orderCh)
	// if err != nil {
	// 	return errors.Join(r.conn.Close(), err)
	// }

	r.notifyCh, err = r.conn.Channel()
	if err != nil {
		return errors.Join(r.conn.Close(), err)
	}
	err = r.notifyCh.ExchangeDeclare(
		"notifications_fanout", // name
		"fanout",               // type
		true,                   // durable
		false,                  // auto-deleted
		false,                  // internal
		false,                  // no-wait
		nil,                    // arguments
	)
	if err != nil {
		return err
	}
	return nil
}
