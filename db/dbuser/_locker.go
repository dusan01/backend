package dbuser

import (
  "hybris/db"
  "sync"
)

var mutexes = map[db.Id]*sync.Mutex{}

type Locker struct {
  sync.Mutex
  user *User
}

func NewLocker(id db.Id) (Locker, error) {
  if _, ok := mutexes[id]; !ok {
    mutexes[id] = &sync.Mutex{}
  }
  mutexes[id].Lock()
  user, err := Get(db.Query{
    "_id": id,
  })
  return Locker{
    user: user,
  }, err
}

func (l Locker) GetUser() User {
  return *l.user
}

func (l Locker) ChangeUsername(username string) Locker {
  l.Lock()
  defer l.Unlock()
  l.user.Username = username
  return l
}

func (l Locker) ChangeDisplayName(displayName string) Locker {
  l.Lock()
  defer l.Unlock()
  l.user.DisplayName = displayName
  return l
}

func (l Locker) ChangeEmail(email string) Locker {
  l.Lock()
  defer l.Unlock()
  l.user.Email = email
  return l
}

func (l Locker) ChangePassword(password []byte) Locker {
  l.Lock()
  defer l.Unlock()
  l.user.Password = password
  return l
}

func (l Locker) ChangeGlobalRole(globalRole int) Locker {
  l.Lock()
  defer l.Unlock()
  l.user.GlobalRole = globalRole
  return l
}

func (l Locker) AddPoints(points int) Locker {
  l.Lock()
  defer l.Unlock()
  l.user.Points += points
  return l
}

func (l Locker) RemovePoints(points int) Locker {
  l.Lock()
  defer l.Unlock()
  l.user.Points -= points
  return l
}

func (l Locker) AddDiamonds(diamonds int) Locker {
  l.Lock()
  defer l.Unlock()
  l.user.Diamonds += diamonds
  return l
}

func (l Locker) RemoveDiamonds(diamonds int) Locker {
  l.Lock()
  defer l.Unlock()
  l.user.Diamonds -= diamonds
  return l
}

func (l Locker) Finish() (err error) {
  defer mutexes[l.user.Id].Unlock()
  err = l.user.Save()
  l.user = nil
  return
}
