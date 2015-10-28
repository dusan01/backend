package db

import (
  "fmt"
  "github.com/gorilla/securecookie"
  "gopkg.in/mgo.v2/bson"
  "hybris/debug"
  "time"
)

type Session struct {
  // Global Ban Id
  Id bson.ObjectId `json:"id" bson:"_id"`

  // 'auth' Cookie value
  Cookie string `json:"cookie" bson:"cookie"`

  // User Id this session belongs to
  UserId bson.ObjectId `json:"userId" bson:"userId"`

  // Date it expires
  Expires *time.Time `json:"expires" bson:"expires"`

  // The date this objects was created
  Created time.Time `json:"created" bson:"created"`

  // The date this object was updated last
  Updated time.Time `json:"updated" bson:"updated"`
}

func NewSession(id bson.ObjectId) (*Session, error) {
  cookie := fmt.Sprintf("%x", securecookie.GenerateRandomKey(64))

  if session, err := GetSession(bson.M{"cookie": cookie}); err == nil {
    return session, nil
  }

  go debug.Log("Creating session for user: [%s]", id)

  return &Session{
    Id:      bson.NewObjectId(),
    Cookie:  fmt.Sprintf("%x", cookie),
    UserId:  id,
    Expires: nil,
    Created: time.Now(),
    Updated: time.Now(),
  }, nil
}

func GetSession(query interface{}) (*Session, error) {
  var s Session
  err := DB.C("sessions").Find(query).One(&s)
  return &s, err
}

func (s Session) Save() error {
  go debug.Log("Saving session: [{\n\t'cookie': %s,\n\t'user': %s\n}]", s.Cookie, s.UserId)
  s.Updated = time.Now()
  _, err := DB.C("sessions").UpsertId(s.Id, s)
  return err
}

func (s Session) Delete() error {
  return DB.C("sessions").RemoveId(s.Id)
}
