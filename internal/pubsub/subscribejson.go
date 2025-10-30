package pubsub

import (
	"encoding/json"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Acktype int

const (
	Ack Acktype = iota
	NackRequeue
	NackDiscard
)

func SubscribeJSON[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType,
	handler func(T) Acktype,
) error {
	amqpChannel, _, err := DeclareAndBind(conn, exchange, queueName, key, queueType)
	if err != nil {
		return err
	}

	deliverChan, err := amqpChannel.Consume(queueName, "", false, false, false, false, nil)
	if err != nil {
		return err
	}

	go func() {
		for msg := range deliverChan {
			var data T

			err := json.Unmarshal(msg.Body, &data)
			if err != nil {
				return
			}

			acktype := handler(data)
			switch acktype {
			case Ack:
				msg.Ack(false)
				log.Println("Message acknowledged")
			case NackRequeue:
				msg.Nack(false, true)
				log.Println("Message requeued")
			case NackDiscard:
				msg.Nack(false, false)
				log.Println("Message discarded")
			}
		}
	}()

	return nil
}
