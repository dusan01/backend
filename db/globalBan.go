package db

import (
  "code.google.com/p/go-uuid/uuid"
  "gopkg.in/mgo.v2"
  "gopkg.in/mgo.v2/bson"
  "strings"
  "sync"
  "time"
)

type GlobalBan struct {
  sync.Mutex

  // Global Ban Id
  Id string `json:"id" bson:"id"`

  // Bannee Id
  // See /db/user/id
  BanneeId string `json:"baneeId" bson:"baneeId"`

  // Banner Id
  // See /db/user/id
  BannerId string `json:"bannerId" bson:"bannerId"`

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

func NewGlobalBan(bannee, banner, reason string, duration int) GlobalBan {
  var t *time.Time
  if duration <= 0 {
    t = nil
  } else {
    ti := time.Now().Add(time.Duration(duration) * time.Second)
    t = &ti
  }
  return GlobalBan{
    Id:       strings.Replace(uuid.NewUUID().String(), "-", "", -1),
    BanneeId: bannee,
    BannerId: banner,
    Reason:   reason,
    Until:    t,
    Created:  time.Now().Format(time.RFC3339),
    Updated:  time.Now().Format(time.RFC3339),
  }
}

func GetGlobalBan(query interface{}) (*GlobalBan, error) {
  var gb GlobalBan
  err := DB.C("globalBans").Find(query).One(&gb)
  return &gb, err
}

func (gb GlobalBan) Save() error {
  err := DB.C("globalBans").Update(bson.M{"id": gb.Id}, gb)
  if err == mgo.ErrNotFound {
    return DB.C("globalBans").Insert(gb)
  }
  return err
}

func (gb GlobalBan) Delete() error {
  return DB.C("globalBans").Remove(bson.M{"id": gb.Id})
}
