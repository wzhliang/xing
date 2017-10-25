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
	RPCTTL = "1"

	// Threshold
	MinHeatbeat = 3
)
