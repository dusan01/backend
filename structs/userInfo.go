package structs

import (
  "gopkg.in/mgo.v2/bson"
  "time"
)

type UserInfo struct {
  Id          bson.ObjectId `json:"id"`
  Username    string        `json:"username"`
  DisplayName string        `json:"displayName"`
  GlobalRole  int           `json:"globalRole"`
  Points      int           `json:"points"`
  Created     time.Time     `json:"createdAt"`
  Updated     time.Time     `json:"updatedAt"`
}
