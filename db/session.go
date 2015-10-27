package db

import (
  "code.google.com/p/go-uuid/uuid"
  "fmt"
  "golang.org/x/crypto/bcrypt"
  "gopkg.in/mgo.v2"
  "gopkg.in/mgo.v2/bson"
  "hybris/debug"
  "strings"
  "sync"
  "time"
)

type Session struct {
  sync.Mutex

  // Primary Key -- Id of the session
  // Not used
  Id string `json:"id" bson:"id"`

  // 'auth' Cookie value
  Cookie string `json:"cookie" bson:"cookie"`

  // User Id this session belongs to
  UserId string `json:"userId" bson:"userId"`

  // Date it expires
  Expires *time.Time `json:"expires" bson:"expires"`

  // Date it was created
  Created string `json:"created" bson:"created"`

  // Date it was last updated
  Updated string `json:"updated" bson:"updated"`
}

func NewSession(id string) (*Session, error) {
  cookie, err := bcrypt.GenerateFromPassword([]byte(strings.Replace(uuid.NewUUID().String(), "-", "", -1)), 10)
  if err != nil {
    return nil, err
  }

  if session, err := GetSession(bson.M{"cookie": cookie}); err == nil {
    return session, nil
  }

  go debug.Log("Creating session for user: [%s]", id)

  return &Session{
    Id:      strings.Replace(uuid.NewUUID().String(), "-", "", -1),
    Cookie:  fmt.Sprintf("%x", cookie),
    UserId:  id,
    Expires: nil,
    Created: time.Now().Format(time.RFC3339),
    Updated: time.Now().Format(time.RFC3339),
  }, nil
}

func GetSession(query interface{}) (*Session, error) {
  var s Session
  err := DB.C("sessions").Find(query).One(&s)
  return &s, err
}

func (s Session) Save() error {
  go debug.Log("Saving session: [{\n\t'cookie': %s,\n\t'user': %s\n}]", s.Cookie, s.UserId)
  err := DB.C("sessions").Update(bson.M{"id": s.Id}, s)
  if err == mgo.ErrNotFound {
    return DB.C("sessions").Insert(s)
  }
  return err
}

func (s Session) Delete() error {
  return DB.C("sessions").Remove(bson.M{"id": s.Id})
}
