package clientaction

import (
	"hybris/realtime"
)

type Client interface {
	Lock()
	Unlock()
	Send([]byte)
	Terminate()
	GetRealtimeUser() *realtime.User
}
