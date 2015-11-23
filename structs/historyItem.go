package structs

import (
	"gopkg.in/mgo.v2/bson"
)

type HistoryItem struct {
	Dj    bson.ObjectId     `json:"dj"`
	Media ResolvedMediaInfo `json:"media"`
	Votes VoteCount         `json:"votes"`
}
