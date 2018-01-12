package xing

// Constants
const (
	// Type
	Command = "command"
	Event   = "event"
	Result  = "result"

	// Event
	Register = "Register"

	// Exchanges
	RPCExchange   = "xing.rpc"
	EventExchange = "xing.event"

	// Client Types
	ProducerClient      = "producer"
	ServiceClient       = "service"
	EventHandlerClient  = "event_handler"
	StreamHandlerClient = "stream_handler"

	// Defaults
	RPCTTL         = int64(1)
	EVTTTL         = int64(15 * 60 * 1000)     // 15 minutes
	STRMTTL        = int64(60 * 1000)          // 1 minutes
	ResultQueueTTL = int64(10 * 60 * 1000)     // 10 minutes
	QueueTTL       = int64(3 * 60 * 60 * 1000) // 3 hours

	// Threshold
	MinHeatbeat = 3

	// Threading
	PoolSize = 1000
	NWorker  = 5
)
