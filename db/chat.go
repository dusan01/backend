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

  // Whether it's a /me message
  Me bool `json:"me" bson:"me"`

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

func NewChat(userId string, communityId string, me bool, message string) Chat {
  if len(message) > 255 {
    message = message[:255]
  }

  return Chat{
    Id:          strings.Replace(uuid.NewUUID().String(), "-", "", -1),
    UserId:      userId,
    CommunityId: communityId,
    Me:          me,
    Message:     message,
    Deleted:     false,
    DeleterId:   "",
    Created:     time.Now().Format(time.RFC3339),
    Updated:     time.Now().Format(time.RFC3339),
  }
}

func (c Chat) Struct() structs.Chat {
  return structs.Chat{
    Id:      c.Id,
    UserId:  c.UserId,
    Me:      c.Me,
    Message: c.Message,
    Time:    c.Created,
  }
}

func GetChat(query interface{}) (*Chat, error) {
  var c Chat
  err := DB.C("chat").Find(query).One(&c)
  return &c, err
}

func (c Chat) Save() error {
  err := DB.C("chat").Update(bson.M{"id": c.Id}, c)
  if err == mgo.ErrNotFound {
    return DB.C("chat").Insert(c)
  }
  return err
}

// Soft delete
func (c Chat) Delete() error {
  c.Deleted = true
  return c.Save()
}
