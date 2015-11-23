package structs

import (
	"gopkg.in/mgo.v2/bson"
)

type Votes struct {
	Woot []bson.ObjectId `json:"woot"`
	Meh  []bson.ObjectId `json:"meh"`
	Save []bson.ObjectId `json:"save"`
}
