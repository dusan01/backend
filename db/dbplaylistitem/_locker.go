package dbplaylistitem

import (
  "hybris/db"
  "sync"
)

var mutexes = map[db.Id]*sync.Mutex{}

type Locker struct {
  sync.Mutex
  playlistItem *PlaylistItem
}

func NewLocker(id db.Id) (Locker, error) {
  if _, ok := mutexes[id]; !ok {
    mutexes[id] = &sync.Mutex{}
  }
  mutexes[id].Lock()
  playlistItem, err := Get(db.Query{
    "_id": id,
  })
  return Locker{
    playlistItem: playlistItem,
  }, err
}

func (l Locker) GetPlaylistItem() PlaylistItem {
  return *l.playlistItem
}

func (l Locker) ChangePlaylistId(playlistId db.Id) Locker {
  l.Lock()
  defer l.Unlock()
  l.playlistItem.PlaylistId = playlistId
  return l
}

func (l Locker) ChangeTitle(title string) Locker {
  l.Lock()
  defer l.Unlock()
  l.playlistItem.Title = title
  return l
}

func (l Locker) ChangeArtist(artist string) Locker {
  l.Lock()
  defer l.Unlock()
  l.playlistItem.Artist = artist
  return l
}

func (l Locker) ChangeMediaId(mediaId db.Id) Locker {
  l.Lock()
  defer l.Unlock()
  l.playlistItem.MediaId = mediaId
  return l
}

func (l Locker) ChangeOrder(order int) Locker {
  l.Lock()
  defer l.Unlock()
  l.playlistItem.Order = order
  return l
}

func (l Locker) Finish() (err error) {
  defer mutexes[l.playlistItem.Id].Unlock()
  err = l.playlistItem.Save()
  l.playlistItem = nil
  return
}
