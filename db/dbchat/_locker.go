package dbchat

import (
  "hybris/db"
  "sync"
)

var mutexes = map[db.Id]*sync.Mutex{}

type Locker struct {
  sync.Mutex
  chat *Chat
}

func NewLocker(id db.Id) (Locker, error) {
  if _, ok := mutexes[id]; !ok {
    mutexes[id] = &sync.Mutex{}
  }
  mutexes[id].Lock()
  chat, err := Get(db.Query{
    "_id": id,
  })
  return Locker{
    chat: chat,
  }, err
}

func (l Locker) GetChat() Chat {
  return *l.chat
}

func (l Locker) ChangeUserId(userId db.Id) Locker {
  l.Lock()
  defer l.Unlock()
  l.chat.UserId = userId
  return l
}

func (l Locker) ChangeCommunityId(communityId db.Id) Locker {
  l.Lock()
  defer l.Unlock()
  l.chat.CommunityId = communityId
  return l
}

func (l Locker) ChangeMe(me bool) Locker {
  l.Lock()
  defer l.Unlock()
  l.chat.Me = me
  return l
}

func (l Locker) ChangeMessage(message string) Locker {
  l.Lock()
  defer l.Unlock()
  l.chat.Message = message
  return l
}

func (l Locker) ChangeDeleted(deleted bool) Locker {
  l.Lock()
  defer l.Unlock()
  l.chat.Deleted = deleted
  return l
}

func (l Locker) ChangeDeleterId(deleterId db.Id) Locker {
  l.Lock()
  defer l.Unlock()
  l.chat.DeleterId = deleterId
  return l
}

func (l Locker) Finish() (err error) {
  defer mutexes[l.chat.Id].Unlock()
  err = l.chat.Save()
  l.chat = nil
  return
}
