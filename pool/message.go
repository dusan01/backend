package pool

import (
  "encoding/json"
)

type Message interface {
  Dispatch(Sender)
}

type Sender interface {
  Send([]byte)
}

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
