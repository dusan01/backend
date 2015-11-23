package dbmute

import (
  "hybris/db"
  "sync"
)

var mutexes = map[db.Id]*sync.Mutex{}

type Locker struct {
  sync.Mutex
  mute *Mute
}

func NewLocker(id db.Id) (Locker, error) {
  if _, ok := mutexes[id]; !ok {
    mutexes[id] = &sync.Mutex{}
  }
  mutexes[id].Lock()
  mute, err := Get(db.Query{
    "_id": id,
  })
  return Locker{
    mute: mute,
  }, err
}

func (l Locker) GetMute() Mute {
  return *l.mute
}

func (l Locker) ChangeMuteeId(muteeId db.Id) Locker {
  l.Lock()
  defer l.Unlock()
  l.mute.MuteeId = muteeId
  return l
}

func (l Locker) ChangeMuterId(muterId db.Id) Locker {
  l.Lock()
  defer l.Unlock()
  l.mute.MuterId = muterId
  return l
}

func (l Locker) ChangeReason(reason string) Locker {
  l.Lock()
  defer l.Unlock()
  l.mute.Reason = reason
  return l
}

func (l Locker) ChangeUntil(until *time.Time) Locker {
  l.Lock()
  defer l.Unlock()
  l.mute.Until = until
  return l
}

func (l Locker) Finish() (err error) {
  defer mutexes[l.mute.Id].Unlock()
  err = l.mute.Save()
  l.mute = nil
  return
}
