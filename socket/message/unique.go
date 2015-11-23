package message

import (
  "encoding/json"
)

type Unique struct {
  Data interface{}
}

func NewUnique(data interface{}) Unique {
  return Unique{data}
}

func (u Unique) Dispatch(s Sender) {
  payload, err := json.Marshal(u.Data)
  if err != nil {
    return
  }

  s.Send(payload)
}
