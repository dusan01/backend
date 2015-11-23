package db

import (
	"gopkg.in/mgo.v2/bson"
	"hybris/structs"
	"time"
)

type Mute struct {
	// Mute Id
	Id bson.ObjectId `json:"id" bson:"id"`

	// Muteee Id
	// See /db/user/id
	MuteeId bson.ObjectId `json:"muteeId" bson:"muteeId"`

	// Muter Id
	// See /db/user/id
	MuterId bson.ObjectId `json:"muterId" bson:"muterId"`

	// Community Id
	// See /db/community/id
	CommunityId bson.ObjectId `json:"communityId" bson:"communityId"`

	// Reason for the mute
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

func NewMute(mutee, muter, community bson.ObjectId, reason string, until *time.Time) *Mute {
	return &Mute{
		Id:          bson.NewObjectId(),
		MuteeId:     mutee,
		MuterId:     muter,
		CommunityId: community,
		Reason:      reason,
		Until:       until,
		Created:     time.Now(),
		Updated:     time.Now(),
	}
}

func GetMute(query interface{}) (*Mute, error) {
	var m Mute
	err := DB.C("mutes").Find(query).One(&m)
	return &m, err
}

func (m Mute) Save() error {
	m.Updated = time.Now()
	_, err := DB.C("mutes").UpsertId(m.Id, m)
	return err
}

func (m Mute) Delete() error {
	return DB.C("mutes").RemoveId(m.Id)
}

func (m Mute) Struct() structs.Mute {
	return structs.Mute{
		Mutee:  m.MuteeId,
		Muter:  m.MuterId,
		Reason: m.Reason,
		Until:  m.Until,
	}
}
