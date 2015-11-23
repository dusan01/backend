package dbchat

import (
  "hybris/db"
  "hybris/structs"
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
  coll, err := db.Session.Collection("media")
  if err != nil && err != uppdb.ErrCollectionDoesNotExists {
    panic(err)
  }
  collection = coll
}

type Chat struct {
  // Database object id
  Id bson.ObjectId `json:"id" bson:"_id"`

  // User this chat belongs to
  UserId bson.ObjectId `json:"userId" bson:"userId"`

  // Community this chat belongs to
  CommunityId bson.ObjectId `json:"communityId" bson:"communityId"`

  // Determines whether or not the message is in italics
  Me bool `json:"me" bson:"me"`

  // Chat message
  // Validation
  //  Max 300 characters
  Message string `json:"message" bson:"message"`

  // Determines whether or not the chat has been deleted
  Deleted bool `json:"deleted" bson:"deleted"`

  // User who deleted the chat
  DeleterId bson.ObjectId `json:"deleterId" bson:"deleterId,omitempty"`

  // When the object was created
  Created time.Time `json:"created" bson:"created"`

  // When the object was last updated
  Updated time.Time `json:"updated" bson:"updated"`
}

func New(userId, communityId bson.ObjectId, me bool, message string) (Chat, error) {
  if len(message) > 300 {
    message = message[:300]
  }

  return Chat{
    Id:          bson.NewObjectId(),
    UserId:      userId,
    CommunityId: communityId,
    Me:          me,
    Message:     message,
    Deleted:     false,
    DeleterId:   "",
    Created:     time.Now(),
    Updated:     time.Now(),
  }, nil
}

func Get(query interface{}) (Chat, error) {
  c, err := get(query)
  if c == nil {
    return Chat{}, err
  }
  return *c, err
}

func get(query interface{}) (*Chat, error) {
  var chat *Chat
  if err := collection.Find(query).One(&chat); err != nil {
    return nil, err
  }
  return getId(chat.Id)
}

func GetId(id bson.ObjectId) (Chat, error) {
  c, err := getId(id)
  if c == nil {
    return Chat{}, err
  }
  return *c, err
}

func getId(id bson.ObjectId) (*Chat, error) {
  if _, ok := getMutexes[id]; !ok {
    getMutexes[id] = &sync.Mutex{}
  }

  getMutexes[id].Lock()
  defer getMutexes[id].Unlock()

  if chat, found := cache.Get(string(id)); found {
    return chat.(*Chat), nil
  }

  var chat *Chat

  if err := collection.Find(uppdb.Cond{"_id": id}).One(&chat); err != nil {
    return nil, err
  }

  cache.Set(string(id), chat, gocache.DefaultExpiration)

  return chat, nil
}

func GetMulti(max int, query interface{}) (chats []Chat, err error) {
  q := collection.Find(query)
  if max < 0 {
    err = q.All(&chats)
  } else {
    err = q.Limit(uint(max)).All(&chats)
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

func LockGet(id bson.ObjectId) (*Chat, error) {
  Lock(id)
  return getId(id)
}

func (c Chat) Save() (err error) {
  c.Updated = time.Now()
  _, err = collection.Append(c)
  return
}

func (c Chat) Delete() error {
  cache.Delete(string(c.Id))
  return collection.Find(uppdb.Cond{"_id": c.Id}).Remove()
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

func StructMulti(chats []Chat) (payload []structs.Chat) {
  for _, c := range chats {
    payload = append(payload, c.Struct())
  }
  return
}
