package xing

// Constants
const (
	// Type
	Command = "command"
	Event   = "event"
	Task    = "task"
	Result  = "result"

	// Event
	Register = "Register"

	// Exchanges
	RPCExchange   = "xing.rpc"
	TaskExchange  = "xing.tasks"
	EventExchange = "xing.event"

	// Client Types
	ProducerClient     = "producer"
	ServiceClient      = "service"
	TaskRunnerClient   = "task_runner"
	EventHandlerClient = "event_handler"

	// Defaults
	RPCTTL         = int64(1)
	EVTTTL         = int64(15 * 60 * 1000)     // 10 minutes
	ResultQueueTTL = int64(10 * 60 * 1000)     // 10 minutes
	QueueTTL       = int64(3 * 60 * 60 * 1000) // 3 hours

	// Threshold
	MinHeatbeat = 3
)
