package db

import (
  "code.google.com/p/go-uuid/uuid"
  "gopkg.in/mgo.v2"
  "gopkg.in/mgo.v2/bson"
  "hybris/structs"
  "strings"
  "sync"
  "time"
)

type PlaylistItem struct {
  sync.Mutex

  // Playlist Item Id
  Id string `json:"id"`

  // Playlist Id
  // See /db/playlist/id
  PlaylistId string `json:"playlistId"`

  // Title
  // The media title
  Title string `json:"title"`

  // Artist
  // The media artist
  Artist string `json:"artist"`

  // Media Id
  // The media id
  MediaId string `json:"mediaId"`

  // Order
  // The order in the playlist
  Order int `json:"order"`

  // The date this objects was created in RFC 3339
  Created string `json:"created"`

  // The date this object was updated last in RFC 3339
  Updated string `json:"updated"`
}

func NewPlaylistItem(playlistId, title, artist, mediaId string) PlaylistItem {
  return PlaylistItem{
    Id:         strings.Replace(uuid.NewUUID().String(), "-", "", -1),
    PlaylistId: playlistId,
    Title:      title,
    Artist:     artist,
    MediaId:    mediaId,
    Order:      -1,
    Created:    time.Now().Format(time.RFC3339),
    Updated:    time.Now().Format(time.RFC3339),
  }
}

func GetPlaylistItem(query interface{}) (*PlaylistItem, error) {
  var pi PlaylistItem
  err := DB.C("communities").Find(query).One(&pi)
  return &pi, err
}

func StructPlaylistItems(items []PlaylistItem) structs.PlaylistItems {
  var payload structs.PlaylistItems
  for _, item := range items {
    payload = append(payload, (&item).Struct())
  }
  return payload
}

func (pi PlaylistItem) Struct() structs.PlaylistItem {
  media, err := GetMedia(bson.M{"mediaid": pi.MediaId})
  if err != nil {
    return structs.PlaylistItem{}
  }
  return structs.PlaylistItem{
    PlaylistId: pi.PlaylistId,
    Title:      pi.Title,
    Artist:     pi.Artist,
    Order:      pi.Order,
    Media:      media.Struct(),
  }
}

func (pi PlaylistItem) Save() error {
  err := DB.C("playlistItems").Update(bson.M{"id": pi.Id}, pi)
  if err == mgo.ErrNotFound {
    return DB.C("playlistItems").Insert(pi)
  }
  return err
}
