package dbcommunityhistory

import (
	"hybris/db"
	"sync"
)

var mutexes = map[db.Id]*sync.Mutex{}

type Locker struct {
	sync.Mutex
	communityHistory *CommunityHistory
}

func NewLocker(id db.Id) (Locker, error) {
	if _, ok := mutexes[id]; !ok {
		mutexes[id] = &sync.Mutex{}
	}
	mutexes[id].Lock()
	communityHistory, err := Get(db.Query{
		"_id": id,
	})
	return Locker{
		communityHistory: communityHistory,
	}, err
}

func (l Locker) GetCommunityHistory(id db.Id) CommunityHistory {
	return *l.communityHistory
}

func (l Locker) ChangeCommunityId(communityId db.Id) Locker {
	l.Lock()
	defer l.Unlock()
	l.communityHistory.CommunityId = communityId
	return l
}

func (l Locker) ChangeUserId(userId db.Id) Locker {
	l.Lock()
	defer l.Unlock()
	l.communityHistory.UserId = userId
	return l
}

func (l Locker) ChangeMediaId(mediaId db.Id) Locker {
	l.Lock()
	defer l.Unlock()
	l.communityHistory.MediaId = mediaId
	return l
}

func (l Locker) ChangeTitle(title string) Locker {
	l.Lock()
	defer l.Unlock()
	l.communityHistory.Title = title
	return l
}

func (l Locker) ChangeArtist(artist string) Locker {
	l.Lock()
	defer l.Unlock()
	l.communityHistory.Artist = artist
	return l
}

func (l Locker) ChangeWoots(woots int) Locker {
	l.Lock()
	defer l.Unlock()
	l.communityHistory.Woots = woots
	return l
}

func (l Locker) ChangeMehs(mehs int) Locker {
	l.Lock()
	defer l.Unlock()
	l.communityHistory.Mehs = mehs
	return l
}

func (l Locker) ChangeGrabs(grabs int) Locker {
	l.Lock()
	defer l.Unlock()
	l.communityHistory.Grabs = grabs
	return l
}

func (l Locker) Finish() (err error) {
	defer mutexes[l.communityHistory.Id].Unlock()
	err = l.communityHistory.Save()
	l.communityHistory = nil
	return
}
