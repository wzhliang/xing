package xing

import (
	"context"
	"crypto/tls"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

// Rules:
// Routing key format:
//     domain.name.instance.type.xxx
// For RPC result it'll be
//     domain.name.instance.result.command

// TODO:
// - check usage of autoAck
// - turn on exclusive for all?
// - hide AMQP details like amqp.Delivery from interface

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
		c.setID(id.InstanceID())
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

// SetTLSConfig ...
func SetTLSConfig(cfg *tls.Config) ClientOpt {
	return func(c *Client) {
		c.tlsConfig = cfg
	}
}

// SetRegistrator ...
func SetRegistrator(reg Registrator) ClientOpt {
	return func(c *Client) {
		c.registrator = reg
	}
}

// SetHealthChecker ...
func SetHealthChecker(hc HealthChecker) ClientOpt {
	return func(c *Client) {
		c.checker = hc
	}
}

func resultTopicName(who string) string {
	// who has to have 3 segments
	return fmt.Sprintf("%s.result.*", who)
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
	tlsConfig    *tls.Config
	serializer   Serializer
	identifier   Identifier
	rpcCounter   uint
	interests    []string
	typ          string
	handlers     map[string]reflect.Value // key is service::command
	inputs       map[string]reflect.Type
	outputs      map[string]reflect.Type
	svc          map[string]interface{} // key is only service
	registrator  Registrator
	checker      HealthChecker
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

func (c *Client) service() string {
	return strings.Join(strings.Split(c.name, ".")[0:2], ".")
}

func (c *Client) setID(id string) {
	arr := strings.Split(c.name, ".")
	arr[2] = id
	c.name = strings.Join(arr, ".")
}

func (c *Client) userType(topic string) string {
	return strings.Split(topic, ".")[4]
}

func (c *Client) respTo(topic string) string {
	return strings.Split(topic, ".")[4]
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
	return c.typ == ServiceClient || c.typ == TaskRunnerClient || c.typ == EventHandlerClient
}

func (c *Client) isService() bool {
	return c.typ == ServiceClient
}

// Register ...
func (c *Client) Register(address string, port int, tags map[string]string, ttl time.Duration) error {
	if c.registrator == nil || c.checker == nil {
		log.Warnf("Need registrator and healthchecker to register.")
		return fmt.Errorf("Invalid configuration")
	}
	if ttl < MinHeatbeat*time.Second {
		ttl = MinHeatbeat * time.Second
	}
	go func() {
		for {
			svc := &Service{
				Name:     c.service(),
				Instance: c.instance(),
				Address:  address,
				Port:     port,
				Tags:     tags,
			}
			if c.checker.Healthy() {
				for tries := 3; tries > 0; tries-- {
					err := c.registrator.Register(svc, ttl)
					if err == nil {
						break
					}
					log.Warnf("Retrying...")
				}
			} else {
				for tries := 3; tries > 0; tries-- {
					err := c.registrator.Deregister(svc)
					if err == nil {
						break
					}
					log.Warnf("Retrying...")
				}
			}
			time.Sleep(ttl - 1*time.Second)
		}
	}()
	return nil
}

func (c *Client) toResult(d *amqp.Delivery) (typ string, v interface{}, err error) {
	typ = c.respTo(d.RoutingKey)
	log.Infof("response to: %s -> %v", typ, c.outputs[typ])
	v = reflect.New(c.outputs[typ]).Interface()
	err = c.serializer.Unmarshal(d.Body, v)
	return
}

func (c *Client) _send(ex string, key string, corrid string, typ string, payload interface{}) error {
	pl, err := c.serializer.Marshal(payload)
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
	if typ == Command {
		msg.Expiration = RPCTTL
	}

	log.Printf("Sending to %s on %s with %s, type: %s", ex, key, corrid, typ)
	return c.ch.Publish(ex, key, false, false, msg)
}

func (c *Client) send(target string, _type string, event string, payload interface{}) error {
	var cor string
	if _type == Command {
		c.m.Lock()
		c.rpcCounter++
		c.m.Unlock()
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
	var key string
	if topicLength(target) == 2 { // load balanced rpc
		key = fmt.Sprintf("%s.*.%s.%s", target, _type, event)
	} else if topicLength(target) == 3 {
		key = fmt.Sprintf("%s.%s.%s", target, _type, event)
	} else {
		return fmt.Errorf("Invalid target: %s", target)
	}
	return c._send(ex, key, cor, event, payload)
}

// Notify ...
func (c *Client) Notify(target string, event string, payload interface{}) error {
	return c.send(target, Event, event, payload)
}

func (c *Client) newChannel() error {
	c.m.Lock()
	c.ch.Close()
	ch, err := c.conn.Channel()
	c.ch = ch
	c.m.Unlock()
	if err != nil {
		return err
	}
	return nil
}

// Call ...
func (c *Client) Call(ctx context.Context, target string, method string, payload interface{}, sync bool) (string, interface{}, error) {
	errCh := make(chan error, 1)
	msgCh := make(chan amqp.Delivery, 1)
	go func() {
		var err error
		var msgs <-chan amqp.Delivery
		if sync {
			err := c.newChannel() // FIXME: ugly hack
			if err != nil {
				errCh <- err
				return
			}
			msgs, err = c.ch.Consume(c.resultQueue.Name, "", false, false, false, false, nil)
			if err != nil {
				errCh <- err
				return
			}
			err = c.ch.Qos(1, 0, false)
			if err != nil {
				errCh <- err
				return
			}
		}
		err = c.send(target, Command, method, payload)
		if err != nil {
			errCh <- err
			return
		}
		if sync {
			for m := range msgs {
				if c.corrid() == m.CorrelationId {
					m.Ack(false)
					msgCh <- m
					return
				}
			}
		}
	}()

	if !sync {
		return "", nil, nil
	}

	select {
	case msg := <-msgCh:
		return c.toResult(&msg)
	case err := <-errCh:
		log.Errorf("operation %s failed: %v", method, err)
		return "", nil, err
	case <-ctx.Done():
		err := fmt.Errorf("RPC timeout: %s", method)
		return "", nil, err
	}
}

// RunTask called by producer to start a task
func (c *Client) RunTask(target string, method string, payload interface{}) (string, error) {
	if topicLength(target) != 3 {
		return "", fmt.Errorf("invalid target: %s", target)
	}
	err := c.send(target, Task, method, payload)
	if err != nil {
		return "", err
	}

	return c.taskid(), nil
}

// WaitForTask ...
func (c *Client) WaitForTask(taskID string) (string, interface{}, error) {
	err := c.newChannel() // FIXME: ugly hack
	if err != nil {
		return "", nil, err
	}
	msgs, err := c.ch.Consume(c.resultQueue.Name, "", false, false, false, false, nil)
	if err != nil {
		return "", nil, err
	}
	for m := range msgs {
		if taskID == m.CorrelationId {
			m.Ack(false)
			return c.toResult(&m)
		}
	}
	return "", nil, fmt.Errorf("wtf?")
}

// Respond called by RPC server or task runner
func (c *Client) Respond(delivery amqp.Delivery, command string, payload interface{}) error {
	key := fmt.Sprintf("%s.%s.%s", delivery.ReplyTo, Result, command)
	return c._send(delivery.Exchange, key, delivery.CorrelationId, command, payload)
}

// Close ...
func (c *Client) Close() {
	_, err := c.ch.QueueDelete(c.resultQueue.Name, false, false, false)
	if err != nil {
		log.Warnf("Error deleting queue: %v", err)
	}
	c.conn.Close()
}

func (c *Client) connect() (*amqp.Connection, error) {
	if c.tlsConfig != nil {
		return amqp.DialTLS(c.url, c.tlsConfig)
	}
	return amqp.Dial(c.url)
}

func bootStrap(name string, url string, opts ...ClientOpt) (*Client, error) {
	c := &Client{
		name:       fmt.Sprintf("%s.%s", name, (&RandomIdentifier{}).InstanceID()),
		url:        url,
		serializer: &ProtoSerializer{},
		identifier: &RandomIdentifier{},
		svc:        make(map[string]interface{}),
	}
	// default to events from own domain
	c.interests = []string{fmt.Sprintf("%s.#", c.domain())}
	// handle options
	for _, o := range opts {
		o(c)
	}
	if topicLength(name) == 3 {
		c.name = name // Allow client to specify it's own name
	}
	conn, err := c.connect()
	if err != nil {
		return nil, err
	}
	c.conn = conn

	ch, err := c.conn.Channel()
	if err != nil {
		return nil, err
	}
	c.ch = ch

	err = ch.ExchangeDeclare(EventExchange, "topic", true, true, false, false, nil)
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

// NewClient ...
func NewClient(name string, url string, opts ...ClientOpt) (*Client, error) {
	c, err := bootStrap(name, url, opts...)
	if err != nil {
		return nil, err
	}
	c.typ = ProducerClient
	qn := fmt.Sprintf("xing.C.%s.result", name)
	c.resultQueue, err = c.ch.QueueDeclare(qn, false, false, false, false, nil)
	if err != nil {
		return nil, err
	}
	key := resultTopicName(c.name)
	log.Printf("Subscribing (%s <-> %s) on %s", c.resultQueue.Name, RPCExchange, key)
	err = c.ch.QueueBind(c.resultQueue.Name, key, RPCExchange, false, nil)
	if err != nil {
		return nil, err
	}
	log.Printf("Subscribing (%s <-> %s) on %s", c.resultQueue.Name, TaskExchange, key)
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
	c.typ = ServiceClient
	// Same queue for all services - load balancing
	if topicLength(name) != 2 && topicLength(name) != 3 {
		return nil, fmt.Errorf("Invalid name for service: %s", name)
	}
	svc := fmt.Sprintf("xing.S.svc-%s", name)
	c.serviceQueue, err = c.ch.QueueDeclare(svc, false, false, false, false, nil)
	if err != nil {
		return nil, err
	}
	key := fmt.Sprintf("%s.#", name)
	log.Printf("Subscribing (%s <-> %s) on %s", svc, RPCExchange, key)
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
	c.typ = TaskRunnerClient
	if topicLength(name) != 3 {
		return nil, fmt.Errorf("invalid name for task runner: %s", name)
	}
	svc := fmt.Sprintf("tkr-%s", c.name)
	c.serviceQueue, err = c.ch.QueueDeclare(svc, false, false, false, false, nil)
	if err != nil {
		return nil, err
	}
	key := fmt.Sprintf("%s.%s.*", c.name, Task)
	log.Printf("Subscribing (%s <-> %s) on %s", svc, TaskExchange, key)
	err = c.ch.QueueBind(svc, key, TaskExchange, false, nil)
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
	c.typ = EventHandlerClient
	n := fmt.Sprintf("xing.S.evh-%s", c.name)
	c.queue, err = c.ch.QueueDeclare(n, false, false, false, false, nil)
	if err != nil {
		return nil, err
	}
	for _, key := range c.interests {
		log.Printf("Subscribing (%s <-> %s) on %s", c.queue.Name, EventExchange, key)
		err = c.ch.QueueBind(c.queue.Name, key, EventExchange, false, nil)
		if err != nil {
			return nil, err
		}
	}
	return c, err
}

// NewHandler ...
func (c *Client) NewHandler(service string, v interface{}) {
	c.m.Lock()
	defer c.m.Unlock()
	c.svc[service] = v // save the handler object
	c.handlers, c.inputs, c.outputs = Methods(service, v)
	for name := range c.handlers {
		log.Infof("+++ %s", c.handlers[name])
	}
}

// Run Only valid for service or event handler
func (c *Client) Run() error {
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
		utype := c.userType(d.RoutingKey)
		if c.inputs[utype] == nil {
			log.Infof("Unknown method: %s", utype)
			continue
		}
		log.Infof("method: %s", utype)
		m := reflect.New(c.inputs[utype])
		err := c.serializer.Unmarshal(d.Body, m.Interface())
		if err != nil {
			log.Errorf("Unable to unmarshal message: %s", d.Body)
			log.Info(err)
			continue
		}
		// I know the signature
		resp := reflect.New(c.outputs[utype])
		params := make([]reflect.Value, 0)
		params = append(params, reflect.ValueOf(c.svc[getService(utype)])) // this pointer
		params = append(params, reflect.ValueOf(context.Background()))
		params = append(params, m)
		params = append(params, resp)
		ret := c.handlers[utype].Call(params)
		if !ret[0].IsNil() {
			log.Errorf("RPC [%s] failed: %v", utype, ret[0])
			// FIXME: return error
		}
		if !c.isService() || c.outputs[utype].Name() == "Void" { // magic Void
			log.Info("No response required.")
		} else {
			err = c.Respond(d, utype, resp.Interface())
			if err != nil {
				log.Errorf("Unable to send response: %v", err)
				return err
			}
		}
		d.Ack(false)
	}

	return nil
}

func topicLength(name string) int {
	return len(strings.Split(name, "."))
}
