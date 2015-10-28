package structs

import (
  "gopkg.in/mgo.v2/bson"
  "time"
)

type Chat struct {
  Id      bson.ObjectId `json:"id"`
  UserId  bson.ObjectId `json:"userId"`
  Me      bool          `json:"me"`
  Message string        `json:"message"`
  Time    time.Time     `json:"time"`
}
