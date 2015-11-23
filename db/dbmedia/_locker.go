package dbmedia

import (
  "hybris/db"
  "sync"
)

var mutexes = map[db.Id]*sync.Mutex{}

type Locker struct {
  sync.Mutex
  media *Media
}

func NewLocker(id db.Id) (Locker, error) {
  if _, ok := mutexes[id]; !ok {
    mutexes[id] = &sync.Mutex{}
  }
  mutexes[id].Lock()
  media, err := Get(db.Query{
    "_id": id,
  })
  return Locker{
    media: media,
  }, err
}

func (l Locker) GetMedia() Media {
  return *l.media
}

func (l Locker) ChangeMediaId(mediaId string) Locker {
  l.Lock()
  defer l.Unlock()
  l.media.MediaId = mediaId
  return l
}

func (l Locker) ChangeImage(image string) Locker {
  l.Lock()
  defer l.Unlock()
  l.media.Image = image
  return l
}

func (l Locker) ChangeLength(length int) Locker {
  l.Lock()
  defer l.Unlock()
  l.media.Length = length
  return l
}

func (l Locker) ChangeTitle(title string) Locker {
  l.Lock()
  defer l.Unlock()
  l.media.Title = title
  return l
}

func (l Locker) ChangeArtists(artist string) Locker {
  l.Lock()
  defer l.Unlock()
  l.media.Artist = artist
  return l
}

func (l Locker) Changeblurb(blurb string) Locker {
  l.Lock()
  defer l.Unlock()
  l.media.Blurb = blurb
  return l
}

func (l Locker) AddPlays(plays int) Locker {
  l.Lock()
  defer l.Unlock()
  l.media.Plays += plays
  return l
}

func (l Locker) AddWoots(woots int) Locker {
  l.Lock()
  defer l.Unlock()
  l.media.Woots += woots
  return l
}

func (l Locker) AddMehs(mehs int) Locker {
  l.Lock()
  defer l.Unlock()
  l.media.Mehs += mehs
  return l
}

func (l Locker) AddGrabs(grabs int) Locker {
  l.Lock()
  defer l.Unlock()
  l.media.Grabs += grabs
  return l
}

func (l Locker) AddPlaylists(playlists int) Locker {
  l.Lock()
  defer l.Unlock()
  l.media.Playlists += playlists
  return l
}

func (l Locker) Finish() (err error) {
  defer mutexes[l.media.Id].Unlock()
  err = l.media.Save()
  l.media = nil
  return
}
