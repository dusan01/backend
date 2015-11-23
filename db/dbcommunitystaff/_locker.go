package dbcommunitystaff

import (
  "hybris/db"
  "sync"
)

var mutexes = map[db.Id]*sync.Mutex{}

type Locker struct {
  sync.Mutex
  communityStaff *CommunityStaff
}

func NewLocker(id db.Id) (Locker, error) {
  if _, ok := mutexes[id]; !ok {
    mutexes[id] = &sync.Mutex{}
  }
  mutexes[id].Lock()
  communityStaff, err := Get(db.Query{
    "_id": id,
  })
  return Locker{
    communityStaff: communityStaff,
  }, err
}

func (l Locker) ChangeCommunityId(communityId db.Id) Locker {
  l.Lock()
  defer l.Unlock()
  l.communityStaff.CommunityId = communityId
  return l
}

func (l Locker) ChangeUserId(userId db.Id) Locker {
  l.Lock()
  defer l.Unlock()
  l.communityStaff.UserId = userId
  return l
}

func (l Locker) ChangeRole(role int) Locker {
  l.Lock()
  defer l.Unlock()
  l.communityStaff.Role = role
  return l
}

func (l Locker) Finish() (err error) {
  defer mutexes[l.communityStaff.Id].Unlock()
  err = l.communityStaff.Save()
  l.communityStaff = nil
  return
}
