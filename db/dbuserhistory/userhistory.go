package dbuserhistory

import (
  "hybris/db"
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
  coll, err := db.Session.Collection("userhistory")
  if err != nil && err != uppdb.ErrCollectionDoesNotExists {
    panic(err)
  }
  collection = coll
}

type UserHistory struct {
  // Database object id
  Id bson.ObjectId `json:"id" bson:"_id"`

  // Community this belongs to
  CommunityId bson.ObjectId `json:"communityId" bson:"communityId"`

  // User who was djing
  UserId bson.ObjectId `json:"userId" bson:"userId"`

  // Database media object id
  MediaId bson.ObjectId `json:"mediaId" bson:"mediaId"`

  // Title of the media inherited from PlaylistItem
  Title string `json:"title" bson:"title"`

  // Artist of the media inherited from PlaylistItem
  Artist string `json:"artist" bson:"artist"`

  // Ammount of times people wooted
  Woots int `json:"woots" bson:"woots"`

  // Amount of times people meh'd
  Mehs int `json:"mehs" bson:"mehs"`

  // Amount of times people grabbed
  Grabs int `json:"grabbed" bson:"grabbed"`

  // The date this objects was created
  Created time.Time `json:"created" bson:"created"`

  // The date this object was updated last
  Updated time.Time `json:"updated" bson:"updated"`
}

func New(communityId, userId, mediaId bson.ObjectId) (UserHistory, error) {
  return UserHistory{
    Id:          bson.NewObjectId(),
    CommunityId: communityId,
    UserId:      userId,
    MediaId:     mediaId,
    Woots:       0,
    Mehs:        0,
    Grabs:       0,
    Created:     time.Now(),
    Updated:     time.Now(),
  }, nil
}

func Get(query interface{}) (UserHistory, error) {
  uh, err := get(query)
  if uh == nil {
    return UserHistory{}, err
  }
  return *uh, err
}

func get(query interface{}) (*UserHistory, error) {
  var userHistory *UserHistory
  if err := collection.Find(query).One(&userHistory); err != nil {
    return nil, err
  }
  return getId(userHistory.Id)
}

func GetId(id bson.ObjectId) (UserHistory, error) {
  uh, err := getId(id)
  if uh == nil {
    return UserHistory{}, err
  }
  return *uh, err
}

func getId(id bson.ObjectId) (*UserHistory, error) {
  if _, ok := getMutexes[id]; !ok {
    getMutexes[id] = &sync.Mutex{}
  }

  getMutexes[id].Lock()
  defer getMutexes[id].Unlock()

  if userHistory, found := cache.Get(string(id)); found {
    return userHistory.(*UserHistory), nil
  }

  var userHistory *UserHistory

  if err := collection.Find(uppdb.Cond{"_id": id}).One(&userHistory); err != nil {
    return nil, err
  }

  cache.Set(string(id), userHistory, gocache.DefaultExpiration)

  return userHistory, nil
}

func GetMulti(max int, query interface{}) (userHistory []UserHistory, err error) {
  q := collection.Find(query)
  if max < 0 {
    err = q.All(&userHistory)
  } else {
    err = q.Limit(uint(max)).All(&userHistory)
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

func LockGet(id bson.ObjectId) (*UserHistory, error) {
  Lock(id)
  return getId(id)
}

func (uh UserHistory) Save() (err error) {
  uh.Updated = time.Now()
  _, err = collection.Append(uh)
  return
}

func (uh UserHistory) Delete() error {
  return collection.Find(uppdb.Cond{"_id": uh.Id}).Remove()
}
