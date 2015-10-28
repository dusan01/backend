package structs

import (
  "gopkg.in/mgo.v2/bson"
  "time"
)

type CommunityPlayingInfo struct {
  DjId    bson.ObjectId     `json:"djId"`
  Started time.Time         `json:"started"`
  Media   ResolvedMediaInfo `json:"media"`
  Votes   Votes             `json:"votes"`
}
