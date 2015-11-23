package dbuserhistory

import (
	"hybris/db"
	"sync"
)

var mutexes = map[db.Id]*sync.Mutex{}

type Locker struct {
	sync.Mutex
	userHistory *UserHistory
}

func NewLocker(id db.Id) (Locker, error) {
	if _, ok := mutexes[id]; !ok {
		mutexes[id] = &sync.Mutex{}
	}
	mutexes[id].Lock()
	userHistory, err := Get(db.Query{
		"_id": id,
	})
	return Locker{
		userHistory: userHistory,
	}, err
}

func (l Locker) GetUserHistory(id db.Id) UserHistory {
	return *l.userHistory
}

func (l Locker) ChangeCommunityId(communityId db.Id) Locker {
	l.Lock()
	defer l.Unlock()
	l.userHistory.CommunityId = communityId
	return l
}

func (l Locker) ChangeUserId(userId db.Id) Locker {
	l.Lock()
	defer l.Unlock()
	l.userHistory.UserId = userId
	return l
}

func (l Locker) ChangeMediaId(mediaId db.Id) Locker {
	l.Lock()
	defer l.Unlock()
	l.userHistory.MediaId = mediaId
	return l
}

func (l Locker) ChangeTitle(title string) Locker {
	l.Lock()
	defer l.Unlock()
	l.userHistory.Title = title
	return l
}

func (l Locker) ChangeArtist(artist string) Locker {
	l.Lock()
	defer l.Unlock()
	l.userHistory.Artist = artist
	return l
}

func (l Locker) ChangeWoots(woots int) Locker {
	l.Lock()
	defer l.Unlock()
	l.userHistory.Woots = woots
	return l
}

func (l Locker) ChangeMehs(mehs int) Locker {
	l.Lock()
	defer l.Unlock()
	l.userHistory.Mehs = mehs
	return l
}

func (l Locker) ChangeGrabs(grabs int) Locker {
	l.Lock()
	defer l.Unlock()
	l.userHistory.Grabs = grabs
	return l
}

func (l Locker) Finish() (err error) {
	defer mutexes[l.userHistory.Id].Unlock()
	err = l.userHistory.Save()
	l.userHistory = nil
	return
}
