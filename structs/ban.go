package structs

import (
  "gopkg.in/mgo.v2/bson"
  "time"
)

type Ban struct {
  Bannee bson.ObjectId `json:"bannee"`
  Banner bson.ObjectId `json:"banner"`
  Reason string        `json:"reason"`
  Until  *time.Time    `json:"until"`
}
