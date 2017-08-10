package xing

import (
	"fmt"
	"time"

	"github.com/streadway/amqp"
)

// whatever
const (
	DefaultExchange = "wisecloud"
)

type Client struct {
	name       string
	url        string
	conn       *amqp.Connection
	ch         *amqp.Channel
	ownQ       amqp.Queue
	exchange   string
	eventTopic string
}

func (c *Client) Send(target string, payload interface{}) error {
	// Assuming to be string for now
	pl := payload.(string)
	msg := amqp.Publishing{
		DeliveryMode: amqp.Persistent,
		Timestamp:    time.Now(),
		ContentType:  "text/plain",
		Body:         []byte(pl),
	}

	return c.ch.Publish(c.exchange, c.eventTopic, false, false, msg)
}

func (c *Client) Receive() (<-chan amqp.Delivery, error) {
	return c.ch.Consume(
		c.ownQ.Name, // queue
		"",          // consumer
		true,        // auto-ack
		false,       // exclusive
		false,       // no-local
		false,       // no-wait
		nil,         // args
	)
}

func bootStrap(name string, url string) (*Client, error) {
	c := &Client{
		name: name,
		url:  url,
	}
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}
	c.conn = conn
	// TODO: when to close connection?

	ch, err := c.conn.Channel()
	if err != nil {
		return nil, err
	}
	c.ch = ch

	err = ch.ExchangeDeclare(DefaultExchange, "fanout", true, true, false, false, nil)
	if err != nil {
		return nil, err
	}

	c.exchange = DefaultExchange
	c.eventTopic = fmt.Sprintf("%s.event", c.name)

	return c, nil
}

func NewProducer(name string, url string) (*Client, error) {
	return bootStrap(name, url)
}

func NewConsumer(name string, url string) (*Client, error) {
	c, err := bootStrap(name, url)
	if err != nil {
		return nil, err
	}
	c.ownQ, err = c.ch.QueueDeclare("whatever", false, false, false, false, nil)
	if err != nil {
		return nil, err
	}
	err = c.ch.QueueBind(c.ownQ.Name, c.eventTopic, c.exchange, false, nil)
	if err != nil {
		return nil, err
	}
	return c, err
}
