package db

import (
  "code.google.com/p/go-uuid/uuid"
  "errors"
  "gopkg.in/mgo.v2"
  "gopkg.in/mgo.v2/bson"
  "hybris/structs"
  "strings"
  "sync"
  "time"
)

type Playlist struct {
  sync.Mutex

  // Playist Id
  Id string `json:"id"`

  // Playlist name
  // Validation
  //  1-30 Characters
  Name string `json:"name"`

  // /db/user/id
  OwnerId string `json:"ownerId"`

  // Selected
  // Whether or not the playlist is selected
  // Validation
  //  Only one playlist can be selected at a time
  Selected bool `json:"selected"`

  // The date this objects was created in RFC 3339
  Created string `json:"created"`

  // The date this object was updated last in RFC 3339
  Updated string `json:"updated"`
}

func NewPlaylist(name, ownerId string, selected bool) (*Playlist, error) {
  if length := len(name); length < 1 || length > 30 {
    return nil, errors.New("Name is invalid")
  }

  return &Playlist{
    Id:       strings.Replace(uuid.NewUUID().String(), "-", "", -1),
    Name:     name,
    OwnerId:  ownerId,
    Selected: selected,
    Created:  time.Now().Format(time.RFC3339),
    Updated:  time.Now().Format(time.RFC3339),
  }, nil
}

func GetPlaylist(query interface{}) (*Playlist, error) {
  var p Playlist
  err := DB.C("playlists").Find(query).One(&p)
  return &p, err
}

func StructPlaylists(playlists []Playlist) []structs.PlaylistInfo {
  var payload []structs.PlaylistInfo
  for _, p := range playlists {
    payload = append(payload, (&p).Struct())
  }
  return payload
}

func (p *Playlist) Struct() structs.PlaylistInfo {
  items, err := p.GetItems()
  if err != nil {
    return structs.PlaylistInfo{}
  }

  return structs.PlaylistInfo{
    Name:     p.Name,
    Id:       p.Id,
    OwnerId:  p.OwnerId,
    Selected: p.Selected,
    Length:   len(items),
  }
}

func (p *Playlist) Save() error {
  err := DB.C("playlists").Update(bson.M{"id": p.Id}, p)
  if err == mgo.ErrNotFound {
    return DB.C("playlists").Insert(p)
  }
  return err
}

func (p *Playlist) Delete() error {
  return DB.C("playlists").Remove(bson.M{"id": p.Id})
}

func (p *Playlist) Select(u *User) error {
  playlists, err := u.GetPlaylists()
  if err != nil {
    return nil
  }

  for _, playlist := range playlists {
    playlist.Selected = (playlist.Id == p.Id)
    if err := playlist.Save(); err != nil {
      return err
    }
  }
  return nil
}

func (p *Playlist) GetItems() ([]PlaylistItem, error) {
  var items []PlaylistItem
  err := DB.C("playlistItems").Find(bson.M{"playlistid": p.Id}).Iter().All(&items)
  if err != nil {
    return items, err
  }
  return p.sortItems(items), err
}

func (p *Playlist) SaveItems(items []PlaylistItem) error {
  items = p.recalculateItems(items)
  for _, item := range items {
    if err := item.Save(); err != nil {
      return err
    }
  }
  return nil
}

func (p *Playlist) sortItems(items []PlaylistItem) []PlaylistItem {
  payload := make([]PlaylistItem, len(items))
  for _, v := range items {
    payload[v.Order] = v
  }
  return payload
}

func (p *Playlist) recalculateItems(items []PlaylistItem) []PlaylistItem {
  payload := make([]PlaylistItem, len(items))
  for i, item := range items {
    item.Order = i
    payload = append(payload, item)
  }
  return payload
}
