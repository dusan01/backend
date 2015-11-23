package message

import (
	"encoding/json"
)

type Action struct {
	Id     string      `json:"i"`
	Status int         `json:"s"`
	Action string      `json:"a"`
	Data   interface{} `json:"d"`
}

func NewAction(id string, status int, action string, data interface{}) Action {
	return Action{id, status, action, data}
}

func (a Action) Dispatch(s Sender) {
	payload, err := json.Marshal(a)
	if err != nil {
		return
	}

	s.Send(payload)
}
