package xing

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/streadway/amqp"
	"github.com/wzhliang/xing/utils"
)

// Rules:
// Routing key format:
//     domain.name.instance.type.xxx
// For RPC result it'll be
//     domain.name.instance.result.command

// TODO:
// - hide AMQP details like amqp.Delivery from interface
// - should we do something about undelivered response?
//   e.g. rabbitmq crashed when we're sending RPC response

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

// SetBrokerTimeout set a timeout for a server to limit broker reconnect
// attempts. count: number of retries; interval: number of seconds between retries
func SetBrokerTimeout(count, interval int) ClientOpt {
	return func(c *Client) {
		c.retryCount = count
		c.retryInterval = interval
	}
}

func resultTopicName(who string) string {
	// who has to have 3 segments
	return fmt.Sprintf("%s.result.*", who)
}

// Client ...
type Client struct {
	sync.Mutex
	name          string
	url           string
	conn          *amqp.Connection
	ch            *amqp.Channel // incoming channel
	sch           *amqp.Channel // outgoing channel
	queue         amqp.Queue    // queue for consumption, for RPC client, it holds results
	tlsConfig     *tls.Config
	serializer    Serializer
	identifier    Identifier
	rpcCounter    uint
	interests     []string
	typ           string
	handlers      map[string]reflect.Value // key is service::command
	inputs        map[string]reflect.Type
	outputs       map[string]reflect.Type
	svc           map[string]interface{} // key is only service
	registrator   Registrator
	checker       HealthChecker
	watchStop     chan bool
	retryCount    int
	retryInterval int
	single        bool
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
		log.Warn().Msg("Need registrator and healthchecker to register.")
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
					log.Warn().Msg("Retrying...")
				}
			} else {
				for tries := 3; tries > 0; tries-- {
					err := c.registrator.Deregister(svc)
					if err == nil {
						break
					}
					log.Warn().Msg("Retrying...")
				}
			}
			time.Sleep(ttl - 1*time.Second)
		}
	}()
	return nil
}

func (c *Client) toResult(d *amqp.Delivery) (typ string, v interface{}, err error) {
	typ = c.respTo(d.RoutingKey)
	log.Info().Str("type", typ).Msgf("response %v", c.outputs[typ])
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

	err = c.newSendChannel()
	if err != nil {
		return err
	}
	log.Info().Str("to", ex).Str("key", key).Str("corid", corrid).Str("type", typ).
		Msg("Sending")
	return c.sch.Publish(ex, key, false, false, msg)
}

func (c *Client) send(target string, _type string, event string, payload interface{}) error {
	var cor string
	if _type == Command {
		c.Lock()
		c.rpcCounter++
		c.Unlock()
		cor = c.corrid()
	} else if _type == Task {
		c.rpcCounter++
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

func (c *Client) newSendChannel() error {
	err := c.setup()
	if err != nil {
		return err
	}
	if c.sch != nil {
		return nil
	}
	c.sch.Close()
	ch, err := c.conn.Channel()
	c.Lock()
	c.sch = ch
	c.Unlock()
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) newChannel() error {
	err := c.setup()
	if err != nil {
		return err
	}
	ch, err := c.conn.Channel()
	c.Lock()
	if c.ch != nil {
		c.ch.Close()
	}
	c.ch = ch
	c.Unlock()
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
			err = c.newChannel()
			if err != nil {
				errCh <- err
				return
			}
			msgs, err = c.ch.Consume(
				c.queue.Name, // queue
				"",           // consumer
				true,         // autoack
				c.single,     // exclusive
				false,        // noLocal
				false,        // noWait
				nil,
			)
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
		log.Error().Str("method", method).Err(err).Msg("Operation failed")
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
	msgs, err := c.ch.Consume(c.queue.Name, "", false, false, false, false, nil)
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
	if c.conn == nil {
		log.Warn().Msg("connection closed already")
		return
	}
	c.stopWatch()
	if c.ch != nil {
		_, err := c.ch.QueueDelete(c.queue.Name, false, false, false)
		if err != nil {
			log.Warn().Err(err).Msg("Error deleting queue")
		}
	}
	c.conn.Close()
}

func (c *Client) connect() (*amqp.Connection, error) {
	if c.tlsConfig != nil {
		return amqp.DialTLS(c.url, c.tlsConfig)
	}
	return amqp.Dial(c.url)
}

// watches the connection
func (c *Client) watch() {
	errors := make(chan *amqp.Error)
	c.conn.NotifyClose(errors)

	for {
		select {
		case err := <-errors:
			log.Warn().Err(err).Msg("Connection lost")
			c.Lock()
			c.conn = nil
			c.sch = nil
			c.ch = nil
			c.Unlock()
			return
		case stop := <-c.watchStop:
			if stop {
				return
			}
		}
	}
}

func (c *Client) stopWatch() {
	c.watchStop <- true
}

func (c *Client) setup() error {
	if c.conn != nil {
		return nil
	}
	conn, err := c.connect()
	if err != nil {
		log.Error().Msgf("connection failed: %v", err)
		return err
	}
	c.Lock()
	c.conn = conn
	c.Unlock()

	if c.ch != nil {
		c.ch.Close()
	}
	ch, err := c.conn.Channel()
	if err != nil {
		log.Error().Msgf("creating channel failed: %v", err)
		return err
	}
	c.Lock()
	c.ch = ch
	c.Unlock()

	ch, err = c.conn.Channel()
	if err != nil {
		log.Error().Msgf("creating channel failed: %v", err)
		return err
	}
	c.Lock()
	c.sch = ch
	c.Unlock()

	err = ch.ExchangeDeclare(
		EventExchange, // name
		"topic",       // kind
		true,          // durable
		false,         // autodelete
		false,         // internal
		false,         // noWait
		nil,
	)
	if err != nil {
		return err
	}

	err = ch.ExchangeDeclare(
		RPCExchange, // name
		"topic",     // kind
		true,        // durable
		false,       // autodelete
		false,       // internal
		false,       // noWait
		nil,
	)
	if err != nil {
		return err
	}

	err = ch.ExchangeDeclare(
		TaskExchange, // name
		"topic",      // kind
		true,         // durable
		false,        // autodelete
		false,        // internal
		false,        // noWait
		nil,
	)
	if err != nil {
		return err
	}
	err = ch.Qos(1, 0, false)
	if err != nil {
		return err
	}

	go c.watch()
	return nil
}

func (c *Client) setupClient() error {
	if c.conn != nil {
		log.Info().Msg("nothing to do")
		return nil
	}
	err := c.setup()
	if err != nil {
		log.Error().Err(err).Msg("setup failed")
		return err
	}
	qn := fmt.Sprintf("xing.C.%s.result", c.name)
	c.queue, err = c.ch.QueueDeclare(
		qn,    // name
		true,  // durable
		false, // autoDelete
		false, // exclusive
		false, // noWait
		amqp.Table{
			"x-expires":     ResultQueueTTL,
			"x-message-ttl": RPCTTL,
		}, // args
	)
	if err != nil {
		log.Error().Err(err).Msg("setup failed")
		return err
	}
	key := resultTopicName(c.name)
	log.Info().Str("queue", c.queue.Name).Str("exchange", RPCExchange).Str("key", key).
		Msg("Subscribing")
	err = c.ch.QueueBind(c.queue.Name, key, RPCExchange, false, nil)
	if err != nil {
		log.Error().Err(err).Msg("setup failed")
		return err
	}
	log.Info().Str("queue", c.queue.Name).Str("exchange", TaskExchange).Str("key", key).
		Msg("Subscribing")
	err = c.ch.QueueBind(c.queue.Name, key, TaskExchange, false, nil)
	if err != nil {
		log.Error().Err(err).Msg("setup failed")
		return err
	}
	return nil
}

func (c *Client) setupService() error {
	if c.conn != nil {
		return nil
	}
	err := c.setup()
	if err != nil {
		return err
	}
	var name string
	if c.single {
		name = c.name
	} else {
		name = c.service()
	}
	qn := fmt.Sprintf("xing.S.svc-%s", name)
	c.queue, err = c.ch.QueueDeclare(
		qn,    // name
		true,  // durable
		false, // autoDelete
		false, // exclusive
		false, // noWait
		amqp.Table{
			"x-expires":     QueueTTL,
			"x-message-ttl": RPCTTL,
		}, // args
	)
	if err != nil {
		return err
	}
	key := fmt.Sprintf("%s.#", name)
	log.Info().Str("queue", c.queue.Name).Str("exchange", RPCExchange).Str("key", key).Msg("Subscribing")
	err = c.ch.QueueBind(c.queue.Name, key, RPCExchange, false, nil)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) setupEventHandler() error {
	err := c.setup()
	if err != nil {
		return err
	}
	qn := fmt.Sprintf("xing.S.evh-%s", c.name)
	c.queue, err = c.ch.QueueDeclare(
		qn,    // name
		true,  // durable
		false, // autoDelete
		false, // exclusive
		false, // noWait
		amqp.Table{
			"x-expires": QueueTTL,
		}, // args
	)
	if err != nil {
		return err
	}
	for _, key := range c.interests {
		log.Info().Str("queue", c.queue.Name).Str("exchange", EventExchange).Str("key", key).Msg("Subscribing")
		err = c.ch.QueueBind(c.queue.Name, key, EventExchange, false, nil)
		if err != nil {
			return err
		}
	}
	return nil
}

// common for both client and server
func bootStrap(name string, url string, opts ...ClientOpt) (*Client, error) {
	c := &Client{
		name:          fmt.Sprintf("%s.%s", name, (&RandomIdentifier{}).InstanceID()),
		url:           url,
		serializer:    &ProtoSerializer{},
		identifier:    &RandomIdentifier{},
		svc:           make(map[string]interface{}),
		watchStop:     make(chan bool),
		retryCount:    30,
		retryInterval: 1,
		rpcCounter:    uint(utils.Random(1000, 9999)),
		single:        true,
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

	return c, nil
}

// NewClient ...
// FIXME: should restrict client name to be 3 segments
func NewClient(name string, url string, opts ...ClientOpt) (*Client, error) {
	c, err := bootStrap(name, url, opts...)
	if err != nil {
		return nil, err
	}
	c.typ = ProducerClient
	err = c.setupClient()
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
	if topicLength(name) == 2 {
		c.single = false
	}

	err = c.setupService()
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
	c.queue, err = c.ch.QueueDeclare(svc, false, false, false, false, nil)
	if err != nil {
		return nil, err
	}
	key := fmt.Sprintf("%s.%s.*", c.name, Task)
	log.Info().Str("queue", c.queue.Name).Str("exchange", TaskExchange).Str("key", key).
		Msg("Subscribing")
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
	err = c.setupEventHandler()
	if err != nil {
		return nil, err
	}

	return c, err
}

// NewHandler ...
func (c *Client) NewHandler(service string, v interface{}) {
	c.Lock()
	defer c.Unlock()
	c.svc[service] = v // save the handler object
	c.handlers, c.inputs, c.outputs = Methods(service, v)
	for name := range c.handlers {
		log.Info().Str("name", c.handlers[name].String()).Msg("+++ ")
	}
}

// Run Only valid for service or event handler
func (c *Client) _run() error {
	if !c.isConsumer() {
		panic("only consumer can start a server")
	}
	var err error
	if c.isService() {
		err = c.setupService()
	} else {
		err = c.setupEventHandler()
	}
	if err != nil {
		return fmt.Errorf("unable to connect to server")
	}
	log.Info().Str("type", c.typ).Msg("server")
	msgs, err := c.ch.Consume(
		c.queue.Name, // queue
		c.name,       // consumer
		true,         // autoAck,
		c.single,     // exclusive
		false,        // noLocal
		false,        // noWait
		nil,
	)
	if err != nil {
		log.Error().Msgf("consume failed: %v", err)
		return err
	}

	for d := range msgs {
		utype := c.userType(d.RoutingKey)
		if c.inputs[utype] == nil {
			log.Info().Str("method", utype).Msg("Unknown method")
			continue
		}
		log.Info().Str("method", utype).Msg("Handling")
		m := reflect.New(c.inputs[utype])
		err := c.serializer.Unmarshal(d.Body, m.Interface())
		if err != nil {
			log.Error().Bytes("msg", d.Body).Err(err).Msg("Unable to unmarshal message")
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
			log.Error().Str("method", utype).Msg("RPC failed")
			// FIXME: return error
		}
		if !c.isService() || c.outputs[utype].Name() == "Void" { // magic Void
			log.Info().Msg("No response required.")
		} else {
			err = c.Respond(d, utype, resp.Interface())
			if err != nil {
				log.Error().Err(err).Msg("Unable to send response")
				return err
			}
		}
	}

	return nil
}

// Run ...
func (c *Client) Run() error {
	var err error
	retry := c.retryCount
	for retry > 0 {
		log.Info().Msg("starting server...")
		err = c._run()
		if err == nil {
			retry = c.retryCount
		} else {
			retry--
		}
		time.Sleep(time.Duration(c.retryInterval) * time.Second)
	}
	return err
}

func topicLength(name string) int {
	return len(strings.Split(name, "."))
}

func init() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
}
