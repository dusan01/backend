package dbsession

import (
  "fmt"
  "hybris/db"
  "sync"
  "time"

  "github.com/gorilla/securecookie"
  gocache "github.com/pmylund/go-cache"
  "gopkg.in/mgo.v2/bson"
  uppdb "upper.io/db"
)

var (
  collection  uppdb.Collection
  cache       = gocache.New(db.CacheExpiration, db.CacheCleanupInterval)
  getMutexes  = map[bson.ObjectId]*sync.Mutex{}
  lockMutexes = map[bson.ObjectId]*sync.Mutex{}
)

func init() {
  coll, err := db.Session.Collection("sessions")
  if err != nil && err != uppdb.ErrCollectionDoesNotExists {
    panic(err)
  }
  collection = coll
}

type Session struct {
  // Database object id
  Id bson.ObjectId `json:"id" bson:"_id"`

  // Cookie value for 'auth'
  Cookie string `json:"cookie" bson:"cookie"`

  // The user that this session belongs to
  UserId bson.ObjectId `json:"userId" bson:"userId"`

  // When the session expires
  Expires *time.Time `json:"expires" bson:"expires"`

  // When the obejct was created
  Created time.Time `json:"created" bson:"created"`

  // When the obejct was last updated
  Updated time.Time `json:"updated" bson:"updated"`
}

func New(userId bson.ObjectId) (Session, error) {
  if session, err := Get(uppdb.Cond{"userId": userId}); err == nil {
    return session, nil
  }

  cookie := fmt.Sprintf("%x", securecookie.GenerateRandomKey(64))

  if _, err := Get(uppdb.Cond{"cookie": cookie}); err == nil {
    return New(userId)
  }

  return Session{
    Id:      bson.NewObjectId(),
    Cookie:  cookie,
    UserId:  userId,
    Expires: nil,
    Created: time.Now(),
    Updated: time.Now(),
  }, nil
}

func Get(query interface{}) (Session, error) {
  s, err := get(query)
  if s == nil {
    return Session{}, err
  }
  return *s, err
}

func get(query interface{}) (*Session, error) {
  var session *Session
  if err := collection.Find(query).One(&session); err != nil {
    return nil, err
  }
  return getId(session.Id)
}

func GetId(id bson.ObjectId) (Session, error) {
  s, err := getId(id)
  if s == nil {
    return Session{}, err
  }
  return *s, err
}

func getId(id bson.ObjectId) (*Session, error) {
  if _, ok := getMutexes[id]; !ok {
    getMutexes[id] = &sync.Mutex{}
  }

  getMutexes[id].Lock()
  defer getMutexes[id].Unlock()

  if session, found := cache.Get(string(id)); found {
    return session.(*Session), nil
  }

  var session *Session

  if err := collection.Find(uppdb.Cond{"_id": id}).One(&session); err != nil {
    return nil, err
  }

  cache.Set(string(id), session, gocache.DefaultExpiration)

  return session, nil
}

func GetMulti(max int, query interface{}) (sessions []Session, err error) {
  q := collection.Find(query)
  if max < 0 {
    err = q.All(&sessions)
  } else {
    err = q.Limit(uint(max)).All(&sessions)
  }
  return
}

func Lock(id bson.ObjectId) {
  if _, ok := lockMutexes[id]; !ok {
    lockMutexes[id] = &sync.Mutex{}
  }

  lockMutexes[id].Lock()
}

func Unlock(id bson.ObjectId) {
  if _, ok := lockMutexes[id]; !ok {
    return
  }
  lockMutexes[id].Unlock()
}

func LockGet(id bson.ObjectId) (*Session, error) {
  Lock(id)
  return getId(id)
}

func (s Session) Save() (err error) {
  s.Updated = time.Now()
  _, err = collection.Append(s)
  return
}

func (s Session) Delete() error {
  return collection.Find(uppdb.Cond{"_id": s.Id}).Remove()
}
