package dbsession

import (
	"hybris/db"
	"sync"
	"time"
)

var mutexes = map[db.Id]*sync.Mutex{}

type Locker struct {
	sync.Mutex
	session *Session
}

func NewLocker(id db.Id) (Locker, error) {
	if _, ok := mutexes[id]; !ok {
		mutexes[id] = &sync.Mutex{}
	}
	mutexes[id].Lock()
	session, err := Get(db.Query{
		"_id": id,
	})
	return Locker{
		session: session,
	}, err
}

func (l Locker) GetSession() Session {
	return *l.session
}
func (l Locker) ChangeCookie(cookie string) Locker {
	l.Lock()
	defer l.Unlock()
	l.session.Cookie = cookie
	return l
}

func (l Locker) ChangeUserId(userId db.Id) Locker {
	l.Lock()
	defer l.Unlock()
	l.session.UserId = userId
	return l
}

func (l Locker) ChangeExpiration(expires *time.Time) Locker {
	l.Lock()
	defer l.Unlock()
	l.session.Expires = expires
	return l
}

func (l Locker) Finish() (err error) {
	defer mutexes[l.session.Id].Unlock()
	err = l.session.Save()
	l.session = nil
	return
}
