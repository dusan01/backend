package dbglobalban

import (
	"hybris/db"
	"sync"
	"time"
)

var mutexes = map[db.Id]*sync.Mutex{}

type Locker struct {
	sync.Mutex
	globalBan *GlobalBan
}

func NewLocker(id db.Id) (Locker, error) {
	if _, ok := mutexes[id]; !ok {
		mutexes[id] = &sync.Mutex{}
	}
	mutexes[id].Lock()
	globalBan, err := Get(db.Query{
		"_id": id,
	})
	return Locker{
		globalBan: globalBan,
	}, err
}

func (l Locker) GetGlobalBan() GlobalBan {
	return *l.globalBan
}

func (l Locker) ChangeBanneeId(banneeId db.Id) Locker {
	l.Lock()
	defer l.Unlock()
	l.globalBan.BanneeId = banneeId
	return l
}

func (l Locker) ChangeBannerId(bannerId db.Id) Locker {
	l.Lock()
	defer l.Unlock()
	l.globalBan.BannerId = bannerId
	return l
}

func (l Locker) ChangeReason(reason string) Locker {
	l.Lock()
	defer l.Unlock()
	l.globalBan.Reason = reason
	return l
}

func (l Locker) ChangeUntil(until *time.Time) Locker {
	l.Lock()
	defer l.Unlock()
	l.globalBan.Until = until
	return l
}

func (l Locker) Finish() (err error) {
	defer mutexes[l.globalBan.Id].Unlock()
	err = l.globalBan.Save()
	l.globalBan = nil
	return
}
