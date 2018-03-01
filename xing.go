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
	worker "github.com/wzhliang/xing/pkg/worker"
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

// SetSerializer sets data serializer. Default is protobuf.
func SetSerializer(ser Serializer) ClientOpt {
	return func(c *Client) {
		c.serializer = ser
	}
}

// SetIdentifier sets identifer of the instance. Default is random string
func SetIdentifier(id Identifier) ClientOpt {
	return func(c *Client) {
		c.identifier = id
		c.setID(id.InstanceID())
	}
}

// SetInterets subscribes to events. Must be called for event handler.
func SetInterets(topic ...string) ClientOpt {
	return func(c *Client) {
		for _, t := range topic {
			c.interests = append(c.interests, t)
		}
	}
}

// SetTLSConfig configures TLS connection to a server.
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

// Client is a wrapper struct for amqp client.
type Client struct {
	sync.Mutex
	wg            sync.WaitGroup
	name          string
	url           string
	conn          *amqp.Connection
	tlsConfig     *tls.Config
	serializer    Serializer
	identifier    Identifier
	interests     []string
	workers       [NWorker]*Worker // workers for producer
	ch            *amqp.Channel    // for consumer
	sch           *amqp.Channel
	queue         amqp.Queue
	pool          worker.Dispatcher
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

func (c *Client) isConsumer() bool {
	return c.typ == ServiceClient ||
		c.typ == EventHandlerClient ||
		c.typ == StreamHandlerClient
}

func (c *Client) isService() bool {
	return c.typ == ServiceClient
}

func (c *Client) isEventHandler() bool {
	return c.typ == EventHandlerClient
}

func (c *Client) isStreamHandler() bool {
	return c.typ == StreamHandlerClient
}

// Register starts periodic registration to a database such as etcd or consul.
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

func (c *Client) toResult(d *amqp.Delivery) (v interface{}, err error) {
	typ := c.respTo(d.RoutingKey)
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

	err = c.setup()
	if err != nil {
		return err
	}
	log.Info().Str("to", ex).Str("key", key).Str("corid", corrid).Str("type", typ).
		Msg("Sending")
	return c.sch.Publish(ex, key, false, false, msg)
}

// this should only be called for non-RPC payload
func (c *Client) send(target string, _type string, event string, payload interface{}) error {
	var cor string
	if _type == Command {
		panic("RPC is not supposed to use this function")
	}
	cor = "N/A"

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

// Notify sends an event.
func (c *Client) Notify(target string, event string, payload interface{}) error {
	return c.send(target, Event, event, payload)
}

// ensureConnection makes sure TCP connection is available
// return true if no new connection is made
func (c *Client) ensureConnection() (bool, error) {
	if c.conn != nil {
		return true, nil
	}
	conn, err := c.connect()
	if err != nil {
		c.conn = nil
		return false, err
	}
	log.Info().Str("addr", conn.LocalAddr().String()).Msg("new TCP connection")
	c.Lock()
	c.conn = conn
	c.Unlock()
	go c.watch()

	// send channel is bound to the connection
	ch, err := c.conn.Channel()
	if err != nil {
		return false, err
	}
	c.Lock()
	c.sch = ch
	c.Unlock()
	return false, nil
}

func (c *Client) newChannel() error {
	old, err := c.ensureConnection()
	if err != nil {
		return err
	}

	// new consume channel is always created
	if old && c.ch != nil {
		c.ch.Close()
	}
	ch, err := c.conn.Channel()
	c.Lock()
	c.ch = ch
	c.Unlock()
	if err != nil {
		return err
	}
	return nil
}

// Call invokes an remote method. Should not be called externally.
func (c *Client) Call(ctx context.Context, target string, method string, payload interface{}, sync bool) (interface{}, error) {
	req := Request{
		ctx:     ctx,
		target:  target,
		method:  method,
		payload: payload,
	}
	ret, err := c.pool.Send(req)
	return ret, err
}

// Respond called by RPC server. Should not be called externally.
func (c *Client) Respond(delivery amqp.Delivery, command string, payload interface{}) error {
	key := fmt.Sprintf("%s.%s.%s", delivery.ReplyTo, Result, command)
	return c._send(delivery.Exchange, key, delivery.CorrelationId, command, payload)
}

func (c *Client) closePool() {
	c.pool.Close()
	log.Info().Msg("stopping worker...")
	c.wg.Wait()
	log.Info().Msg("closing workers...")
	for _, w := range c.workers {
		w.Close()
	}
}

// Close shuts down a client or server.
func (c *Client) Close() {
	if c.conn == nil {
		log.Warn().Msg("connection closed already")
		return
	}
	c.stopWatch()
	if c.typ == ProducerClient {
		c.closePool()
	} else {
		if c.ch != nil {
			_, err := c.ch.QueueDelete(c.queue.Name, false, false, false)
			if err != nil {
				log.Warn().Err(err).Msg("Error deleting queue")
			}
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
	log.Info().Str("addr", conn.LocalAddr().String()).Msg("local")
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
	c.pool = worker.NewDispatcher(PoolSize)
	if c.pool == nil {
		return fmt.Errorf("creating pool failed")
	}
	for i := 0; i < NWorker; i++ {
		// ideally not the whole client is passed in
		c.workers[i] = newWorker(c.conn, c, i+1)
		if c.workers[i] == nil {
			return fmt.Errorf("failed to creat worker")
		}
		c.wg.Add(1)
		c.pool.AddWorker(c.workers[i])
	}
	return nil
}

func (c *Client) setupConsumer() error {
	if c.isService() {
		return c.setupService()
	} else if c.isEventHandler() {
		return c.setupEventHandler()
	} else if c.isStreamHandler() {
		return c.setupStreamHandler()
	}
	panic("Unknown consumer type")
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
	// Persistenting queue cause reconnection problem with a load balanced
	// cluster:
	// when a server reconnects, it will (most likely) be connected to a
	// different node the one that's died, declaring a durable queue will fail
	// with 504 error
	c.queue, err = c.ch.QueueDeclare(
		qn,    // name
		false, // durable
		true,  // autoDelete
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
			"x-expires":     QueueTTL,
			"x-message-ttl": EVTTTL,
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

func (c *Client) setupStreamHandler() error {
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
	qn := fmt.Sprintf("xing.S.stm-%s", name)
	c.queue, err = c.ch.QueueDeclare(
		qn,    // name
		true,  // durable
		false, // autoDelete
		false, // exclusive
		false, // noWait
		amqp.Table{
			"x-expires":     QueueTTL,
			"x-message-ttl": STRMTTL,
		}, // args
	)
	if err != nil {
		return err
	}
	c.interests = append(c.interests, fmt.Sprintf("%s.#", name)) // auto sub
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
		single:        true,
	}
	// handle options
	for _, o := range opts {
		o(c)
	}
	if topicLength(name) == 3 {
		c.name = name // Allow client to specify it's own name
	}

	return c, nil
}

// NewClient creates a RPC/event client.
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

// NewService creates a RPC service. If the name has only 2 segments, different
// service instances are balanced.
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

// NewEventHandler creates a new event handler.
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

// NewStreamHandler creates a new data stream handler.
func NewStreamHandler(name string, url string, opts ...ClientOpt) (*Client, error) {
	c, err := bootStrap(name, url, opts...)
	if err != nil {
		return nil, err
	}
	c.typ = StreamHandlerClient
	// Same queue for all services - load balancing
	if topicLength(name) != 2 && topicLength(name) != 3 {
		return nil, fmt.Errorf("Invalid name for stream handler: %s", name)
	}
	if topicLength(name) == 2 {
		c.single = false
	}
	err = c.setupStreamHandler()
	if err != nil {
		return nil, err
	}

	return c, err
}

// NewHandler registers handler of protocol. Called from generated code.
func (c *Client) NewHandler(service string, v interface{}) {
	c.Lock()
	defer c.Unlock()
	c.svc[service] = v // save the handler object
	c.handlers, c.inputs, c.outputs = Methods(service, v)
	for name := range c.handlers {
		log.Info().Str("name", c.handlers[name].String()).Msg("+++ ")
	}
}

func (c *Client) handleMessage(ctx context.Context, d amqp.Delivery) error {
	utype := c.userType(d.RoutingKey)
	if c.inputs[utype] == nil {
		log.Info().Str("method", utype).Msg("Unknown method")
		return nil
	}
	log.Info().Str("method", utype).Msg("Handling")
	m := reflect.New(c.inputs[utype])
	err := c.serializer.Unmarshal(d.Body, m.Interface())
	if err != nil {
		log.Error().Bytes("msg", d.Body).Err(err).Msg("Unable to unmarshal message")
		return nil
	}
	// I know the signature
	resp := reflect.New(c.outputs[utype])
	params := make([]reflect.Value, 0)
	params = append(params, reflect.ValueOf(c.svc[getService(utype)])) // this pointer
	params = append(params, reflect.ValueOf(ctx))
	params = append(params, m)
	params = append(params, resp)
	ret := c.handlers[utype].Call(params)
	if !ret[0].IsNil() {
		log.Error().Str("method", utype).Msg("RPC failed")
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

	return nil
}

// Run Only valid for service or event handler
func (c *Client) _run(ctx context.Context) (bool, error) {
	if !c.isConsumer() {
		panic("only consumer can start a server")
	}
	err := c.setupConsumer()
	if err != nil {
		return false, fmt.Errorf("unable to connect to server")
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
		return false, err
	}

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("somebody is stoppping me")
			return true, nil
		case d, ok := <-msgs:
			if !ok {
				return false, fmt.Errorf("channel closed")
			}
			err = c.handleMessage(ctx, d)
			if err != nil {
				return false, err
			}
		}
	}
}

// Run starts a server with default context
func (c *Client) Run() error {
	return c.RunWithContext(context.Background())
}

// RunWithContext starts a RPC/event server. This is a blocking call.
func (c *Client) RunWithContext(ctx context.Context) error {
	var err error
	retry := c.retryCount
	for retry > 0 {
		log.Info().Msg("starting server...")
		stop, err := c._run(ctx)
		if stop {
			return nil
		}
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
	if os.Getenv("XING_TRACE_ON") == "1" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	} else {
		log.Logger = log.Level(zerolog.WarnLevel).
			Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}
}
