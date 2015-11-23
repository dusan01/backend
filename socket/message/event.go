package message

import (
	"encoding/json"
)

type Event struct {
	Event string      `json:"e"`
	Data  interface{} `json:"d"`
}

func NewEvent(event string, data interface{}) Event {
	return Event{event, data}
}

func (e Event) Dispatch(s Sender) {
	payload, err := json.Marshal(e)
	if err != nil {
		return
	}

	s.Send(payload)
}
