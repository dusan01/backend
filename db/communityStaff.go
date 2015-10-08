package db

import (
  "hybris/structs"
  "sync"
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

func NewCommunityStaff() {}

func StructCommunityStaff(cs []CommunityStaff) []structs.StaffItem {
  var payload []structs.StaffItem
  for _, s := range cs {
    payload = append(payload, (&s).Struct())
  }
  return payload
}

func (cs *CommunityStaff) Struct() structs.StaffItem {
  return structs.StaffItem{
    UserId: cs.UserId,
    Role:   cs.Role,
  }
}
