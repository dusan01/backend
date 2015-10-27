package db

import (
  "code.google.com/p/go-uuid/uuid"
  "gopkg.in/mgo.v2"
  "gopkg.in/mgo.v2/bson"
  "strings"
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

func NewBan(bannee, banner, community, reason string, until *time.Time) *Ban {
  return &Ban{
    Id:          strings.Replace(uuid.NewUUID().String(), "-", "", -1),
    BanneeId:    bannee,
    BannerId:    banner,
    CommunityId: community,
    Reason:      reason,
    Until:       until,
    Created:     time.Now().Format(time.RFC3339),
    Updated:     time.Now().Format(time.RFC3339),
  }
}

func GetBan(query interface{}) (*Ban, error) {
  var b Ban
  err := DB.C("bans").Find(query).One(&b)
  return &b, err
}

func (b Ban) Save() error {
  err := DB.C("bans").Update(bson.M{"id": b.Id}, b)
  if err == mgo.ErrNotFound {
    return DB.C("bans").Insert(b)
  }
  return err
}

func (b Ban) Delete() error {
  return DB.C("bans").Remove(bson.M{"id": b.Id})
}
