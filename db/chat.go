package db

import (
  "sync"
)

type Chat struct {
  sync.Mutex

  // Chat Id
  Id string `json:"id" bson:"id"`

  // User Id
  // See /db/user/id
  UserId string `json:"userId" bson:"userId"`

  // Community Id
  // See /db/community/id
  CommunityId string `json:"communityId" bson:"communityId"`

  // Chat message
  // Validation
  //  Max 300 characters
  Message string `json:"message" bson:"message"`

  // Deleted
  // Whether or not the chat message has been deleted
  Deleted bool `json:"deleted" bson:"deleted"`

  // Deleter Id
  // The person who deleted the message
  // See /db/user/id
  DeleterId string `json:"deleterId" bson:"deleterId"`

  // The date this objects was created in RFC 3339
  Created string `json:"created" bson:"created"`

  // The date this object was updated last in RFC 3339
  Updated string `json:"updated" bson:"updated"`
}
