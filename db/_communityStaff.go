package db

import (
	"gopkg.in/mgo.v2/bson"
	"hybris/structs"
	"time"
)

type CommunityStaff struct {
	// Community Staff Id
	Id bson.ObjectId `json:"id" bson:"_id"`

	// Community Id
	// See /db/community/id
	CommunityId bson.ObjectId `json:"communityId" bson:"communityId"`

	// User Id
	// See /db/user/id
	UserId bson.ObjectId `json:"userId" bson:"userId"`

	// Role
	// See enum/COMMUNITY_ROLES
	Role int `json:"role" bson:"role"`

	// The date this objects was created
	Created time.Time `json:"created" bson:"created"`

	// The date this object was updated last
	Updated time.Time `json:"updated" bson:"updated"`
}

func NewCommunityStaff(community, user bson.ObjectId, role int) *CommunityStaff {
	return &CommunityStaff{
		Id:          bson.NewObjectId(),
		CommunityId: community,
		UserId:      user,
		Role:        role,
		Created:     time.Now(),
		Updated:     time.Now(),
	}
}

func GetCommunityStaff(query interface{}) (*CommunityStaff, error) {
	var cs CommunityStaff
	err := DB.C("communityStaff").Find(query).One(&cs)
	return &cs, err
}

func (cs CommunityStaff) Save() error {
	cs.Updated = time.Now()
	_, err := DB.C("communityStaff").UpsertId(cs.Id, cs)
	return err
}

func (cs CommunityStaff) Delete() error {
	return DB.C("communityStaff").RemoveId(cs.Id)
}

func (cs CommunityStaff) Struct() structs.StaffItem {
	return structs.StaffItem{
		UserId: cs.UserId,
		Role:   cs.Role,
	}
}

func StructCommunityStaff(cs []CommunityStaff) []structs.StaffItem {
	var payload []structs.StaffItem
	for _, s := range cs {
		payload = append(payload, s.Struct())
	}
	return payload
}
