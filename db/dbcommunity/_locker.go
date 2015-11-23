package dbcommunity

import (
  "hybris/db"
  "sync"
)

var mutexes = map[db.Id]*sync.Mutex{}

type Locker struct {
  sync.Mutex
  community *Community
}

func NewLocker(id db.Id) (Locker, error) {
  if _, ok := mutexes[id]; !ok {
    mutexes[id] = &sync.Mutex{}
  }
  mutexes[id].Lock()
  community, err := Get(db.Query{
    "_id": id,
  })
  return Locker{
    community: community,
  }, err
}

func (l Locker) GetCommunity() Community {
  return *l.community
}

func (l Locker) ChangeUrl(url string) Locker {
  l.Lock()
  defer l.Unlock()
  l.community.Url = url
  return l
}

func (l Locker) ChangeName(name string) Locker {
  l.Lock()
  defer l.Unlock()
  l.community.Name = name
  return l
}

func (l Locker) ChangeHostId(hostId db.Id) Locker {
  l.Lock()
  defer l.Unlock()
  l.community.HostId = hostId
  return l
}

func (l Locker) ChangeDescription(description string) Locker {
  l.Lock()
  defer l.Unlock()
  l.community.Description = description
  return l
}

func (l Locker) ChangeWelcomeMessage(welcomeMessage string) Locker {
  l.Lock()
  defer l.Unlock()
  l.community.WelcomeMessage = welcomeMessage
  return l
}

func (l Locker) ChangeWaitlistEnabled(waitlistEnabled bool) Locker {
  l.Lock()
  defer l.Unlock()
  l.community.WaitlistEnabled = waitlistEnabled
  return l
}

func (l Locker) ChangeDjRecycling(djRecycling bool) Locker {
  l.Lock()
  defer l.Unlock()
  l.community.DjRecycling = djRecycling
  return l
}

func (l Locker) ChangeNsfw(nsfw bool) Locker {
  l.Lock()
  defer l.Unlock()
  l.community.Nsfw = nsfw
  return l
}

func (l Locker) Finish() (err error) {
  defer mutexes[l.community.Id].Unlock()
  err = l.community.Save()
  l.community = nil
  return
}
