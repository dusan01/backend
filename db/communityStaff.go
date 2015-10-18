package db

import (
  "code.google.com/p/go-uuid/uuid"
  "gopkg.in/mgo.v2"
  "gopkg.in/mgo.v2/bson"
  "hybris/structs"
  "strings"
  "sync"
  "time"
)

type CommunityStaff struct {
  sync.Mutex

  // Community Staff Id
  Id string `json:"id"`

  // Community Id
  // See /db/community/id
  CommunityId string `json:"communityId"`

  // User Id
  // See /db/user/id
  UserId string `json:"userId"`

  // Role
  // See enum/COMMUNITY_ROLES
  Role int `json:"role"`

  // The date this objects was created in RFC 3339
  Created string `json:"created"`

  // The date this object was updated last in RFC 3339
  Updated string `json:"updated"`
}

func NewCommunityStaff(community string, user string, role int) *CommunityStaff {
  return &CommunityStaff{
    Id:          strings.Replace(uuid.NewUUID().String(), "-", "", -1),
    CommunityId: community,
    UserId:      user,
    Role:        role,
    Created:     time.Now().Format(time.RFC3339),
    Updated:     time.Now().Format(time.RFC3339),
  }
}

func GetCommunityStaff(query interface{}) (*CommunityStaff, error) {
  var cs CommunityStaff
  err := DB.C("communityStaff").Find(query).One(&cs)
  return &cs, err
}

func StructCommunityStaff(cs []CommunityStaff) []structs.StaffItem {
  var payload []structs.StaffItem
  for _, s := range cs {
    payload = append(payload, (&s).Struct())
  }
  return payload
}

func (cs CommunityStaff) Save() error {
  err := DB.C("communityStaff").Update(bson.M{"id": cs.Id}, cs)
  if err == mgo.ErrNotFound {
    return DB.C("communityStaff").Insert(cs)
  }
  return err
}

func (cs CommunityStaff) Delete() error {
  return DB.C("communityStaff").Remove(bson.M{"id": cs.Id})
}

func (cs CommunityStaff) Struct() structs.StaffItem {
  return structs.StaffItem{
    UserId: cs.UserId,
    Role:   cs.Role,
  }
}
