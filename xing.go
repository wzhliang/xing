package xing

import (
	"fmt"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

// Rules:
// Routing key format:
//     domain.name.instance.type.xxx

// TODO:
// - check usage of autoAck
// - turn on exclusive for all?
// - hide AMQP deatils like amqp.Delivery from interface

// MessageHandler ...
type MessageHandler func(typ string, v interface{}, ctx amqp.Delivery)

// ClientOpt ...
type ClientOpt func(*Client)

// SetSerializer ...
func SetSerializer(ser Serializer) ClientOpt {
	return func(c *Client) {
		c.serializer = ser
	}
}

// SetIdentifier ...
func SetIdentifier(id Identifier) ClientOpt {
	return func(c *Client) {
		c.identifier = id
	}
}

// SetInterets ...
func SetInterets(topic ...string) ClientOpt {
	return func(c *Client) {
		for _, t := range topic {
			c.interests = append(c.interests, t)
		}
	}
}

func resultTopicName(who string) string {
	return fmt.Sprintf("%s.result", who)
}

// Client ...
type Client struct {
	m            sync.Mutex
	name         string
	url          string
	conn         *amqp.Connection
	ch           *amqp.Channel
	queue        amqp.Queue
	serviceQueue amqp.Queue
	resultQueue  amqp.Queue
	serializer   Serializer
	identifier   Identifier
	rpcCounter   uint
	interests    []string
	typ          string
	// TODO: protect against race condition?
}

func (c *Client) exchange(typ string) string {
	if typ == Event {
		return EventExchange
	} else if typ == Task {
		return TaskExchange
	}
	return RPCExchange
}

func (c *Client) domain() string {
	return strings.Split(c.name, ".")[0]
}

func (c *Client) topicLength(name string) int {
	return len(strings.Split(name, "."))
}

func (c *Client) service() string {
	return strings.Join(strings.Split(c.name, ".")[0:2], ".")
}

func (c *Client) userType(topic string) string {
	return strings.Split(topic, ".")[3]
}

func (c *Client) instance() string {
	return strings.Split(c.name, ".")[2]
}

func (c *Client) corrid() string {
	return fmt.Sprintf("rpc-%s-%d", c.name, c.rpcCounter)
}

func (c *Client) taskid() string {
	return fmt.Sprintf("tsk-%s-%d", c.name, c.rpcCounter)
}

func (c *Client) isConsumer() bool {
	return c.typ == Service || c.typ == TaskRunner || c.typ == EventHandler
}

// should only be called by task runners
func (c *Client) register() error {
	return c.send(c.name, Event, Register, c.serializer.DefaultValue())
}

func (c *Client) _send(ex string, key string, corrid string, userType string, payload interface{}) error {
	pl, err := c.serializer.Marshal(userType, payload)
	if err != nil {
		return err
	}
	msg := amqp.Publishing{
		DeliveryMode:  amqp.Persistent,
		Timestamp:     time.Now(),
		ContentType:   c.serializer.ContentType(),
		ReplyTo:       c.name, // hard-coded result routing key
		CorrelationId: corrid,
		Body:          []byte(pl),
	}

	log.Printf("Sending to %s on %s with %s, type: %s", ex, key, corrid, userType)
	return c.ch.Publish(ex, key, false, false, msg)
}

func (c *Client) send(target string, _type string, event string, payload interface{}) error {
	var cor string
	if _type == Command {
		c.rpcCounter++
		log.Printf("rpcCounter: %d", c.rpcCounter)
		cor = c.corrid()
	} else if _type == Task {
		c.rpcCounter++
		log.Printf("rpcCounter: %d", c.rpcCounter)
		cor = c.taskid()
	} else {
		cor = "N/A"
	}

	ex := c.exchange(_type)
	if target == "" {
		target = c.domain() + "." + "x" + "." + "x"
	}
	key := fmt.Sprintf("%s.%s.%s", target, _type, event)
	return c._send(ex, key, cor, event, payload)
}

// NotifyAll ...
func (c *Client) NotifyAll(event string, payload interface{}) error {
	return c.send("", Event, event, payload)
}

// NotifySome ...
func (c *Client) NotifySome(target string, event string, payload interface{}) error {
	return c.send(target, Event, event, payload)
}

func (c *Client) newChannel() error {
	c.ch.Close()
	ch, err := c.conn.Channel()
	if err != nil {
		return err
	}
	c.ch = ch
	return nil
}

// Call ...
func (c *Client) Call(target string, method string, payload interface{}) error {
	err := c.newChannel() // FIXME: ugly hack, should try to resuse the channel
	if err != nil {
		return err
	}
	msgs, err := c.ch.Consume(c.resultQueue.Name, "", false, false, false, false, nil)
	if err != nil {
		return err
	}
	err = c.ch.Qos(1, 0, false)
	if err != nil {
		return err
	}
	err = c.send(target, Command, method, payload)
	if err != nil {
		return err
	}
	for m := range msgs {
		if c.corrid() == m.CorrelationId {
			m.Ack(false)
			break
		}
	}

	return nil
}

// RunTask called by producer to start a task
func (c *Client) RunTask(target string, method string, payload interface{}) (string, error) {
	if c.topicLength(target) != 3 {
		return "", fmt.Errorf("invalid target: %s", target)
	}
	err := c.send(target, Task, method, payload)
	if err != nil {
		return "", err
	}

	return c.taskid(), nil
}

// WaitForTask ...
func (c *Client) WaitForTask(taskID string) (*amqp.Delivery, error) {
	err := c.newChannel() // FIXME: ugly hack
	if err != nil {
		return nil, err
	}
	msgs, err := c.ch.Consume(c.resultQueue.Name, "", false, false, false, false, nil)
	if err != nil {
		return nil, err
	}
	for m := range msgs {
		if taskID == m.CorrelationId {
			m.Ack(false)
			return &m, nil
		}
	}
	return nil, fmt.Errorf("wtf?")
}

// Respond called by RPC server or task runner
func (c *Client) Respond(delivery amqp.Delivery, respType string, payload interface{}) error {
	key := fmt.Sprintf("%s.%s.x", delivery.ReplyTo, Result)
	return c._send(delivery.Exchange, key, delivery.CorrelationId, respType, payload)
}

// Close ...
func (c *Client) Close() {
	c.conn.Close()
}

func bootStrap(name string, url string, opts ...ClientOpt) (*Client, error) {
	c := &Client{
		name:       fmt.Sprintf("%s.%s", name, (&NoneIdentifier{}).InstanceID()),
		url:        url,
		serializer: &PlainSerializer{},
		identifier: &PodIdentifier{},
	}
	if c.topicLength(name) == 3 { // FIXME: maybe we shouldn't allow this
		c.name = name // Allow client to specify it's own name
	}
	// default to events from own domain
	c.interests = []string{fmt.Sprintf("%s.#", c.domain())}
	for _, o := range opts {
		o(c)
	}
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}
	c.conn = conn

	ch, err := c.conn.Channel()
	if err != nil {
		return nil, err
	}
	c.ch = ch

	err = ch.ExchangeDeclare(EventExchange, "fanout", true, true, false, false, nil)
	if err != nil {
		return nil, err
	}

	err = ch.ExchangeDeclare(RPCExchange, "topic", true, true, false, false, nil)
	if err != nil {
		return nil, err
	}

	err = ch.ExchangeDeclare(TaskExchange, "topic", true, true, false, false, nil)
	if err != nil {
		return nil, err
	}

	return c, nil
}

// NewProducer ...
func NewProducer(name string, url string, opts ...ClientOpt) (*Client, error) {
	c, err := bootStrap(name, url, opts...)
	if err != nil {
		return nil, err
	}
	c.typ = Producer
	c.resultQueue, err = c.ch.QueueDeclare("", false, false, false, false, nil)
	if err != nil {
		return nil, err
	}
	key := fmt.Sprintf("%s.#", resultTopicName(c.name))
	log.Printf("Subscribing to %s on %s", key, RPCExchange)
	err = c.ch.QueueBind(c.resultQueue.Name, key, RPCExchange, false, nil)
	if err != nil {
		return nil, err
	}
	log.Printf("Subscribing to %s on %s", key, TaskExchange)
	err = c.ch.QueueBind(c.resultQueue.Name, key, TaskExchange, false, nil)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// NewService ...
func NewService(name string, url string, opts ...ClientOpt) (*Client, error) {
	c, err := bootStrap(name, url, opts...)
	if err != nil {
		return nil, err
	}
	c.typ = Service
	// Same queue for all services - load balancing
	svc := fmt.Sprintf("svc-%s", c.service())
	c.serviceQueue, err = c.ch.QueueDeclare(svc, false, false, false, false, nil)
	if err != nil {
		return nil, err
	}
	key := fmt.Sprintf("%s.#", c.service())
	log.Printf("Subscribing to %s in %s", key, RPCExchange)
	err = c.ch.QueueBind(svc, key, RPCExchange, false, nil)
	if err != nil {
		return nil, err
	}
	return c, err
}

// NewTaskRunner ...
func NewTaskRunner(name string, url string, opts ...ClientOpt) (*Client, error) {
	c, err := bootStrap(name, url, opts...)
	if err != nil {
		return nil, err
	}
	c.typ = TaskRunner
	if c.topicLength(name) != 3 {
		return nil, fmt.Errorf("invalid name for task runner: %s", name)
	}
	svc := fmt.Sprintf("tkr-%s", c.name)
	c.serviceQueue, err = c.ch.QueueDeclare(svc, false, false, false, false, nil)
	if err != nil {
		return nil, err
	}
	key := fmt.Sprintf("%s.%s.*", c.name, Task)
	log.Printf("Subscribing to %s in %s", key, TaskExchange)
	err = c.ch.QueueBind(svc, key, TaskExchange, false, nil)
	if err != nil {
		return nil, err
	}
	err = c.register()
	if err != nil {
		return nil, err
	}
	return c, err
}

// NewEventHandler ...
func NewEventHandler(name string, url string, opts ...ClientOpt) (*Client, error) {
	c, err := bootStrap(name, url, opts...)
	if err != nil {
		return nil, err
	}
	c.typ = EventHandler
	n := fmt.Sprintf("evh-%s", c.name)
	c.queue, err = c.ch.QueueDeclare(n, false, false, false, false, nil)
	if err != nil {
		return nil, err
	}
	for _, key := range c.interests {
		log.Printf("Subscribing to %s in %s", key, EventExchange)
		err = c.ch.QueueBind(c.queue.Name, key, EventExchange, false, nil)
		if err != nil {
			return nil, err
		}
	}
	return c, err
}

// Loop Only valid for consumers
func (c *Client) Loop(handler MessageHandler) error {
	if !c.isConsumer() {
		panic("only consumer can start a server")
	}
	autoAck := c.typ == Event // for event, turn on auto ack
	msgs, err := c.ch.Consume(c.queue.Name, "", false, false, autoAck, false, nil)
	err = c.ch.Qos(1, 0, false)
	if err != nil {
		return err
	}

	for d := range msgs {
		ser := c.serializer
		m, err := ser.Unmarshal(c.userType(d.RoutingKey), d.Body)
		if err != nil {
			log.Errorf("Unable to unmarshal message: d.Body")
			continue
		}
		log.Printf("Received a message: %s from %s", m, d.ReplyTo)
		handler(c.userType(d.RoutingKey), m, d)
		d.Ack(false)
	}

	return nil
}
