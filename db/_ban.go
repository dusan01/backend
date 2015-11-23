package db

import (
	"gopkg.in/mgo.v2/bson"
	"hybris/structs"
	"time"
)

type Ban struct {
	// Ban Id
	Id bson.ObjectId `json:"id" bson:"id"`

	// Bannee Id
	// See /db/user/id
	BanneeId bson.ObjectId `json:"banneeId" bson:"banneeId"`

	// Banner Id
	// See /db/user/id
	BannerId bson.ObjectId `json:"bannerId" bson:"bannerId"`

	// Community Id
	// See /db/community/id
	CommunityId bson.ObjectId `json:"communityId" bson:"communityId"`

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

func NewBan(bannee, banner, community bson.ObjectId, reason string, until *time.Time) *Ban {
	return &Ban{
		Id:          bson.NewObjectId(),
		BanneeId:    bannee,
		BannerId:    banner,
		CommunityId: community,
		Reason:      reason,
		Until:       until,
		Created:     time.Now(),
		Updated:     time.Now(),
	}
}

func GetBan(query interface{}) (*Ban, error) {
	var b Ban
	err := DB.C("bans").Find(query).One(&b)
	return &b, err
}

func (b Ban) Save() error {
	b.Updated = time.Now()
	_, err := DB.C("bans").UpsertId(b.Id, b)
	return err
}

func (b Ban) Delete() error {
	return DB.C("bans").RemoveId(b.Id)
}

func (b Ban) Struct() structs.Ban {
	return structs.Ban{
		Banner: b.BannerId,
		Bannee: b.BanneeId,
		Reason: b.Reason,
		Until:  b.Until,
	}
}
