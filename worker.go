package xing

import (
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/streadway/amqp"
	wk "github.com/wzhliang/xing/pkg/worker"
	"github.com/wzhliang/xing/utils"
)

// Worker ...
type Worker struct {
	id         int
	ch         *amqp.Channel    // incoming channel
	sch        *amqp.Channel    // outgoing channel
	queue      amqp.Queue       // queue for consumption, for RPC client, it holds results
	conn       *amqp.Connection // own copy for detecting reconnection
	d          wk.Dispatcher
	c          *Client // the AMQP client
	rpcCounter uint
}

// SetDispatcher ...
func (w *Worker) SetDispatcher(dis wk.Dispatcher) {
	w.d = dis
}

// Close ... should be called after stopped
func (w *Worker) Close() {
	// queue gets automatically deleted anyway
}

func (w *Worker) corrid() string {
	return fmt.Sprintf("rpc-%s-%02d-%d", w.c.name, w.id, w.rpcCounter)
}

// Loop ...
func (w *Worker) Loop() {
	for {
		req := w.d.Dispatch()
		if req.In == nil {
			break
		}
		var results chan interface{}
		var errs chan error
		ret := func(v interface{}, err error) {
			results <- v
			errs <- err
		}

		results = req.Result

		errs = req.Err
		if w.c.conn == nil {
			log.Info().Msg("connection not available, sleeping a while...")
			time.Sleep(time.Duration(utils.Random(2000, 3000)) * time.Millisecond)
			// prevent mutlple works from trying to reconnect at the same time
			_, err := w.c.ensureConnection()
			if err == nil {
				w.onConnect(w.c.conn)
			} else {
				ret(nil, fmt.Errorf("connection lost"))
				continue
			}
		} else if w.c.conn != w.conn {
			log.Info().Msg("new conection")
			w.onConnect(w.c.conn)
		}
		ret(w.processJob(req))
	}
	log.Info().Int("id", w.id).Msg("I am done")
	w.c.wg.Done() // FIXME: should not have access to wg
}

func newWorker(conn *amqp.Connection, c *Client, id int) *Worker {
	w := Worker{
		id:         id,
		c:          c,
		conn:       nil,
		rpcCounter: uint(utils.Random(1000, 9999)),
	}
	// onConnect() will be called on demand due to the fact that our own conn is
	// different from Client's conn
	// this makes sure that worker always has a consumer to its queue so that
	// the queue will not be TTLed
	return &w
}

func (w *Worker) onConnect(conn *amqp.Connection) error {
	log.Info().Msg("connected")
	var err error
	w.conn = conn // refreshing own copy
	w.ch, err = conn.Channel()
	if err != nil {
		return nil
	}
	w.sch, err = conn.Channel()
	if err != nil {
		return nil
	}
	err = w.ch.ExchangeDeclare(
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
	// I have to be a client
	w.queue, err = w.ch.QueueDeclare(
		// for client queue, we don't care it's name
		// also, changing it will cause crash
		fmt.Sprintf("%s-%02d", w.c.name, w.id), // name
		true,  // durable
		false, // autoDelete
		true,  // exclusive
		false, // noWait
		amqp.Table{
			"x-expires":     ResultQueueTTL,
			"x-message-ttl": RPCTTL,
		}, // args
	)
	if err != nil {
		log.Error().Err(err).Msg("setup failed")
		return nil
	}
	key := resultTopicName(w.queue.Name) // for client, queuename is the client's name
	log.Info().Str("queue", w.queue.Name).Str("exchange", RPCExchange).Str("key", key).
		Msg("Subscribing")
	err = w.ch.QueueBind(w.queue.Name, key, RPCExchange, false, nil)
	if err != nil {
		log.Error().Err(err).Msg("setup failed")
		return err
	}

	return nil
}

func (w *Worker) sendEx(target string, _type string, event string, replyTo string, payload interface{}) error {
	var cor string

	if _type == Command { // FIXME: actuall worker should only handle Command
		w.rpcCounter++
		cor = w.corrid()
	} else {
		cor = "N/A"
	}

	ex := w.c.exchange(_type)
	var key string
	if topicLength(target) == 2 { // load balanced rpc
		key = fmt.Sprintf("%s.*.%s.%s", target, _type, event)
	} else if topicLength(target) == 3 {
		key = fmt.Sprintf("%s.%s.%s", target, _type, event)
	} else {
		return fmt.Errorf("Invalid target: %s", target)
	}
	return w._sendEx(ex, key, cor, event, replyTo, payload)
}

func (w *Worker) _sendEx(ex string, key string, corrid string, typ string, replyTo string, payload interface{}) error {
	pl, err := w.c.serializer.Marshal(payload)
	if err != nil {
		return err
	}
	msg := amqp.Publishing{
		DeliveryMode:  amqp.Persistent,
		Timestamp:     time.Now(),
		ContentType:   w.c.serializer.ContentType(),
		ReplyTo:       replyTo,
		CorrelationId: corrid,
		Body:          []byte(pl),
	}

	log.Info().Int("id", w.id).Str("to", ex).Str("key", key).Str("corid", corrid).Str("type", typ).Str("replyTo", replyTo).
		Msg("Sending")
	return w.sch.Publish(ex, key, false, false, msg)
}

func (w *Worker) newChannel() (*amqp.Channel, error) {
	_, err := w.c.ensureConnection()
	if err != nil {
		return nil, err
	}

	return w.c.conn.Channel()
}

// this deals with a job blockingly
func (w *Worker) processJob(req wk.Request) (interface{}, error) {
	in := req.In.(Request)
	{
		var err error
		var msgs <-chan amqp.Delivery
		// receive
		// this has to be done before sending, otherwise it's too late
		w.ch.Cancel(w.queue.Name, false)
		msgs, err = w.ch.Consume(
			w.queue.Name, // queue
			w.queue.Name, // consumer
			true,         // autoAck
			true,         // exclusive
			false,        // noLocal
			false,        // noWait
			nil,
		)
		if err != nil {
			log.Error().Err(err).Msg("failed to consume")
			return nil, err
		}
		// send
		err = w.sendEx(in.target, Command, in.method, w.queue.Name, in.payload)
		if err != nil {
			return nil, err
		}
		select {
		case msg := <-msgs:
			if w.corrid() != msg.CorrelationId {
				log.Warn().Str("corrid", msg.CorrelationId).Msg("It looks like you're using multiple thread to access a single channel.")
				return nil, fmt.Errorf("unexpected message received")
			}
			res, err := w.c.toResult(&msg)
			log.Info().Int("id", w.id).Str("corrid", w.corrid()).Msg("Response")
			return res, err
		case <-in.ctx.Done():
			log.Error().Int("id", w.id).Str("corrid", w.corrid()).Str("method", in.method).Msg("RPC timeout")
			err := fmt.Errorf("Your RPC has timedout, commander: %s", in.method)
			return nil, err
		}
	}
}
