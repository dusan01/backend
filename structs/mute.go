package structs

import (
	"time"

	"gopkg.in/mgo.v2/bson"
)

type Mute struct {
	Mutee  bson.ObjectId `json:"mutee"`
	Muter  bson.ObjectId `json:"muter"`
	Reason string        `json:"reason"`
	Until  *time.Time    `json:"until"`
}
