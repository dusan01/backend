package dbban

import (
	"hybris/db"
	"sync"
	"time"
)

var mutexes = map[db.Id]*sync.Mutex{}

type Locker struct {
	sync.Mutex
	ban *Ban
}

func NewLocker(id db.Id) (Locker, error) {
	if _, ok := mutexes[id]; !ok {
		mutexes[id] = &sync.Mutex{}
	}
	mutexes[id].Lock()
	ban, err := Get(db.Query{
		"_id": id,
	})
	return Locker{
		ban: ban,
	}, err
}

func (l Locker) GetBan() Ban {
	return *l.ban
}

func (l Locker) ChangeBannee(banneeId db.Id) Locker {
	l.Lock()
	defer l.Unlock()
	l.ban.BanneeId = banneeId
	return l
}

func (l Locker) ChangeBanner(bannerId db.Id) Locker {
	l.Lock()
	defer l.Unlock()
	l.ban.BannerId = bannerId
	return l
}

func (l Locker) ChangeCommunityId(communityId db.Id) Locker {
	l.Lock()
	defer l.Unlock()
	l.ban.CommunityId = communityId
	return l
}

func (l Locker) ChangeReason(reason string) Locker {
	l.Lock()
	defer l.Unlock()
	l.ban.Reason = reason
	return l
}

func (l Locker) ChangeUntil(until *time.Time) Locker {
	l.Lock()
	defer l.Unlock()
	l.ban.Until = until
	return l
}

func (l Locker) Finish() (err error) {
	defer mutexes[l.ban.Id].Unlock()
	err = l.ban.Save()
	l.ban = nil
	return
}
