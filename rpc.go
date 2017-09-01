package xing

import "strings"

// RPC methods format: service::command

func getService(method string) string {
	return strings.Split(method, "::")[0]
}

func fullName(service, command string) string {
	return service + "::" + command
}
