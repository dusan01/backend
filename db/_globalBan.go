package db

import (
	"gopkg.in/mgo.v2/bson"
	"time"
)

type GlobalBan struct {
	// Global Ban Id
	Id bson.ObjectId `json:"id" bson:"_id"`

	// Bannee Id
	// See /db/user/id
	BanneeId bson.ObjectId `json:"baneeId" bson:"baneeId"`

	// Banner Id
	// See /db/user/id
	BannerId bson.ObjectId `json:"bannerId" bson:"bannerId"`

	// Reason for the ban
	// Validation
	//  0-500 Characters
	Reason string `json:"reason" bson:"reason"`

	// Until time
	Until *time.Time `json:"until" bson:"until"`

	// The date this objects was created
	Created time.Time `json:"created" bson:"created"`

	// The date this object was updated last
	Updated time.Time `json:"updated" bson:"updated"`
}

func NewGlobalBan(bannee, banner bson.ObjectId, reason string, duration int) GlobalBan {
	var t *time.Time
	if duration <= 0 {
		t = nil
	} else {
		ti := time.Now().Add(time.Duration(duration) * time.Second)
		t = &ti
	}
	return GlobalBan{
		Id:       bson.NewObjectId(),
		BanneeId: bannee,
		BannerId: banner,
		Reason:   reason,
		Until:    t,
		Created:  time.Now(),
		Updated:  time.Now(),
	}
}

func GetGlobalBan(query interface{}) (*GlobalBan, error) {
	var gb GlobalBan
	err := DB.C("globalBans").Find(query).One(&gb)
	return &gb, err
}

func (gb GlobalBan) Save() error {
	gb.Updated = time.Now()
	_, err := DB.C("globalBans").UpsertId(gb.Id, gb)
	return err
}

func (gb GlobalBan) Delete() error {
	return DB.C("globalBans").RemoveId(gb.Id)
}
