package db

import (
  "sync"
  "time"
)

type Ban struct {
  sync.Mutex

  // Ban Id
  Id string `json:"id" bson:"id"`

  // Bannee Id
  // See /db/user/id
  BanneeId string `json:"banneeId" bson:"banneeId"`

  // Banner Id
  // See /db/user/id
  BannerId string `json:"bannerId" bson:"bannerId"`

  // Community Id
  // See /db/community/id
  CommunityId string `json:"communityId" bson:"communityId"`

  // Reason for the ban
  // Validation
  //  0-500 Characters
  Reason string `json:"reason" bson:"reason"`

  // Until time
  Until *time.Time `json:"until" bson:"until"`

  // The date this objects was created in RFC 3339
  Created string `json:"created" bson:"created"`

  // The date this object was updated last in RFC 3339
  Updated string `json:"updated" bson:"updated"`
}
