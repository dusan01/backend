package pool

import (
  "encoding/json"
)

type Message interface {
  Dispatch(*Client)
}

type Action struct {
  Id     int         `json:"i"`
  Status int         `json:"status"`
  Action string      `json:"a"`
  Data   interface{} `json:"d"`
}

func NewAction(id, status int, action string, data interface{}) Action {
  return Action{id, status, action, data}
}

func (a Action) Dispatch(c *Client) {
  payload, err := json.Marshal(a)
  if err != nil {
    return
  }

  c.Send(payload)
}

type Event struct {
  Event string      `json:"e"`
  Data  interface{} `json:"d"`
}

func NewEvent(event string, data interface{}) Event {
  return Event{event, data}
}

func (e Event) Dispatch(c *Client) {
  payload, err := json.Marshal(e)
  if err != nil {
    return
  }

  c.Send(payload)
}
