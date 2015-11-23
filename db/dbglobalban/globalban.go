package dbglobalban

import (
  "errors"
  "hybris/db"
  "hybris/validation"
  "sync"
  "time"

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
  coll, err := db.Session.Collection("globalBans")
  if err != nil && err != uppdb.ErrCollectionDoesNotExists {
    panic(err)
  }
  collection = coll
}

type GlobalBan struct {
  // Database object id
  Id bson.ObjectId `json:"id" bson:"_id"`

  // User who was banned
  BanneeId bson.ObjectId `json:"baneeId" bson:"baneeId"`

  // User who created this ban
  BannerId bson.ObjectId `json:"bannerId" bson:"bannerId"`

  // Reason for the ban
  Reason string `json:"reason" bson:"reason"`

  // When the ban expires
  Until *time.Time `json:"until" bson:"until"`

  // When the object was created
  Created time.Time `json:"created" bson:"created"`

  // When the object was last updated
  Updated time.Time `json:"updated" bson:"updated"`
}

func New(banneeId, bannerId bson.ObjectId, reason string, until *time.Time) (GlobalBan, error) {
  if !validation.Reason(reason) {
    return GlobalBan{}, errors.New("ivalid reason")
  }
  return GlobalBan{
    Id:       bson.NewObjectId(),
    BanneeId: banneeId,
    BannerId: bannerId,
    Reason:   reason,
    Until:    until,
    Created:  time.Now(),
    Updated:  time.Now(),
  }, nil
}

func Get(query interface{}) (GlobalBan, error) {
  gb, err := get(query)
  if gb == nil {
    return GlobalBan{}, err
  }
  return *gb, err
}

func get(query interface{}) (*GlobalBan, error) {
  var globalBan *GlobalBan
  if err := collection.Find(query).One(&globalBan); err != nil {
    return nil, err
  }
  return getId(globalBan.Id)
}

func GetId(id bson.ObjectId) (GlobalBan, error) {
  gb, err := getId(id)
  if gb == nil {
    return GlobalBan{}, err
  }
  return *gb, err
}

func getId(id bson.ObjectId) (*GlobalBan, error) {
  if _, ok := getMutexes[id]; !ok {
    getMutexes[id] = &sync.Mutex{}
  }

  getMutexes[id].Lock()
  defer getMutexes[id].Unlock()

  if globalBan, found := cache.Get(string(id)); found {
    return globalBan.(*GlobalBan), nil
  }

  var globalBan *GlobalBan

  if err := collection.Find(uppdb.Cond{"_id": id}).One(&globalBan); err != nil {
    return nil, err
  }

  cache.Set(string(id), globalBan, gocache.DefaultExpiration)

  return globalBan, nil
}

func GetMulti(max int, query interface{}) (globalBans []GlobalBan, err error) {
  q := collection.Find(query)
  if max < 0 {
    err = q.All(&globalBans)
  } else {
    err = q.Limit(uint(max)).All(&globalBans)
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

func LockGet(id bson.ObjectId) (*GlobalBan, error) {
  Lock(id)
  return getId(id)
}

func (gb GlobalBan) Save() (err error) {
  gb.Updated = time.Now()
  _, err = collection.Append(gb)
  return
}

func (gb GlobalBan) Delete() error {
  cache.Delete(string(gb.Id))
  return collection.Find(uppdb.Cond{"_id": gb.Id}).Remove()
}
