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
	Producer     = "producer"
	Service      = "service"
	TaskRunner   = "task_runner"
	EventHandler = "event_handler"
)
