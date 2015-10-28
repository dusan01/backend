package structs

import (
  "gopkg.in/mgo.v2/bson"
  "time"
)

type Mute struct {
  Mutee  bson.ObjectId `json:"mutee"`
  Muter  bson.ObjectId `json:"muter"`
  Reason string        `json:"reason"`
  Until  *time.Time    `json:"until"`
}
