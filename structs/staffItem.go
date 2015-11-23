package structs

import "gopkg.in/mgo.v2/bson"

type StaffItem struct {
	UserId bson.ObjectId `json:"userId"`
	Role   int           `json:"role"`
}
