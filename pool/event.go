package pool

import (
  "encoding/json"
)

type Event struct {
  Id     int         `json:"i"`
  Status int         `json:"s"`
  Action string      `json:"a"`
  Data   interface{} `json:"d"`
}

func NewEvent(id, status int, action string, data interface{}) *Event {
  return &Event{id, status, action, data}
}

func (e *Event) Dispatch(c *Client) {
  payload, err := json.Marshal(e)
  if err != nil {
    return
  }

  c.Send(payload)
}
