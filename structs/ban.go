package structs

import (
	"time"

	"gopkg.in/mgo.v2/bson"
)

type Ban struct {
	Bannee bson.ObjectId `json:"bannee"`
	Banner bson.ObjectId `json:"banner"`
	Reason string        `json:"reason"`
	Until  *time.Time    `json:"until"`
}
