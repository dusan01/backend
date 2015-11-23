package db

import (
  "errors"
  "gopkg.in/mgo.v2/bson"
  "hybris/structs"
  "time"
)

type Playlist struct {
  // Playist Id
  Id bson.ObjectId `json:"id" bson:"_id"`

  // Playlist name
  // Validation
  //  1-30 Characters
  Name string `json:"name" bson:"name"`

  // /db/user/id
  OwnerId bson.ObjectId `json:"ownerId" bson:"ownerId"`

  // Selected
  // Whether or not the playlist is selected
  // Validation
  //  Only one playlist can be selected at a time
  Selected bool `json:"selected" bson:"selected"`

  // The order that playlists are displayed in the UI
  Order int `json:"order" bson:"order"`

  // The date this objects was create
  Created time.Time `json:"created" bson:"created"`

  // The date this object was updated last
  Updated time.Time `json:"updated" bson:"updated"`
}

func NewPlaylist(name string, ownerId bson.ObjectId) (*Playlist, error) {
  if length := len(name); length < 1 || length > 30 {
    return nil, errors.New("Name is invalid")
  }

  return &Playlist{
    Id:      bson.NewObjectId(),
    Name:    name,
    OwnerId: ownerId,
    Order:   -1,
    Created: time.Now(),
    Updated: time.Now(),
  }, nil
}

func GetPlaylist(query interface{}) (*Playlist, error) {
  var p Playlist
  err := DB.C("playlists").Find(query).One(&p)
  return &p, err
}

func (p Playlist) Delete() error {
  if err := DB.C("playlistItems").RemoveId(bson.M{"playlistId": p.Id}); err != nil {
    return err
  }
  return DB.C("playlists").RemoveId(p.Id)
}

func (p Playlist) Select(u *User) error {
  playlists, err := u.GetPlaylists()
  if err != nil {
    return err
  }

  for _, playlist := range playlists {
    playlist.Selected = (playlist.Id == p.Id)
  }
  return u.SavePlaylists(playlists)
}

func (p Playlist) GetItems() ([]PlaylistItem, error) {
  var items []PlaylistItem
  err := DB.C("playlistItems").Find(bson.M{"playlistId": p.Id}).Iter().All(&items)
  if err != nil {
    return items, err
  }
  return p.sortItems(items), err
}

func (p Playlist) SaveItems(items []PlaylistItem) error {
  items = p.recalculateItems(items)
  for _, item := range items {
    if err := item.Save(); err != nil {
      return err
    }
  }
  return nil
}

func (p Playlist) sortItems(items []PlaylistItem) []PlaylistItem {
  payload := make([]PlaylistItem, len(items))
  for _, v := range items {
    payload[v.Order] = v
  }
  return payload
}

func (p Playlist) recalculateItems(items []PlaylistItem) []PlaylistItem {
  payload := []PlaylistItem{}
  for i, item := range items {
    item.Order = i
    payload = append(payload, item)
  }
  return payload
}

func (p Playlist) Struct() structs.PlaylistInfo {
  items, err := p.GetItems()
  if err != nil {
    return structs.PlaylistInfo{}
  }

  return structs.PlaylistInfo{
    Name:     p.Name,
    Id:       p.Id,
    OwnerId:  p.OwnerId,
    Selected: p.Selected,
    Order:    p.Order,
    Length:   len(items),
  }
}

func StructPlaylists(playlists []Playlist) []structs.PlaylistInfo {
  var payload []structs.PlaylistInfo
  for _, p := range playlists {
    payload = append(payload, p.Struct())
  }
  return payload
}

func (p Playlist) Save() error {
  p.Updated = time.Now()
  _, err := DB.C("playlists").UpsertId(p.Id, p)
  return err
}
