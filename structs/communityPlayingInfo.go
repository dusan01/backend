package structs

import (
	"time"

	"gopkg.in/mgo.v2/bson"
)

type CommunityPlayingInfo struct {
	DjId    bson.ObjectId     `json:"djId"`
	Started time.Time         `json:"started"`
	Media   ResolvedMediaInfo `json:"media"`
	Votes   Votes             `json:"votes"`
}
