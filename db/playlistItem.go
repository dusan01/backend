package db

import (
  "gopkg.in/mgo.v2/bson"
  "hybris/structs"
  "time"
)

type PlaylistItem struct {
  // Playlist Item Id
  Id bson.ObjectId `json:"id" bson:"_id"`

  // Playlist Id
  // See /db/playlist/id
  PlaylistId bson.ObjectId `json:"playlistId" bson:"playlistId"`

  // Title
  // The media title
  Title string `json:"title" bson:"title"`

  // Artist
  // The media artist
  Artist string `json:"artist" bson:"artist"`

  // Media Id
  // /db/media/id
  MediaId bson.ObjectId `json:"mediaId" bson:"mid"`

  // Order
  // The order in the playlist
  Order int `json:"order" bson:"order"`

  // The date this objects was created in RFC 3339
  Created time.Time `json:"created" bson:"created"`

  // The date this object was updated last in RFC 3339
  Updated time.Time `json:"updated" bson:"updated"`
}

func NewPlaylistItem(playlistId, mediaId bson.ObjectId, title, artist string) PlaylistItem {
  return PlaylistItem{
    Id:         bson.NewObjectId(),
    PlaylistId: playlistId,
    Title:      title,
    Artist:     artist,
    MediaId:    mediaId,
    Order:      -1,
    Created:    time.Now(),
    Updated:    time.Now(),
  }
}

func GetPlaylistItem(query interface{}) (*PlaylistItem, error) {
  var pi PlaylistItem
  err := DB.C("communities").Find(query).One(&pi)
  return &pi, err
}

func (pi PlaylistItem) Save() error {
  pi.Updated = time.Now()
  _, err := DB.C("playlistItems").UpsertId(pi.Id, pi)
  return err
}

func (pi PlaylistItem) Delete() error {
  return DB.C("playlistItems").RemoveId(pi.Id)
}

func (pi PlaylistItem) Struct() structs.PlaylistItem {
  media, err := GetMedia(bson.M{"mid": pi.MediaId})
  if err != nil {
    return structs.PlaylistItem{}
  }
  return structs.PlaylistItem{
    Id:         pi.Id,
    PlaylistId: pi.PlaylistId,
    Title:      pi.Title,
    Artist:     pi.Artist,
    Order:      pi.Order,
    Media:      media.Struct(),
  }
}

func StructPlaylistItems(items []PlaylistItem) structs.PlaylistItems {
  var payload structs.PlaylistItems
  for _, item := range items {
    payload = append(payload, item.Struct())
  }
  return payload
}
