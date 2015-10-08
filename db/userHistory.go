package db

import (
  "code.google.com/p/go-uuid/uuid"
  "strings"
  "sync"
  "time"
)

type UserHistory struct {
  sync.Mutex

  // User History Id
  Id string `json:"id"`

  // user Id
  // The user that this belongs to
  // See /db/user/id
  UserId string `json:"userId"`

  // PlaylistItem Id
  // The id of the playlist item
  // See /db/playliatItem/id
  PlaylistItemId string `json:"playlistItemId"`

  // Media Id
  // The media id
  // See /db/media/id
  MediaId string `json:"mediaId"`

  // Title of the media inherited from PlaylistItem
  // See /db/playlistItem/title
  Title string `json:"title"`

  // Artist of the media inherited from PlaylistItem
  // See /db/playlistItem/artist
  Artist string `json:"artist"`

  // Ammount of times people have wooted this
  Woots int `json:"woots"`

  // Amount of times people have meh'd this
  Mehs int `json:"mehs"`

  // Amount of times people have saved this
  Saves int `json:"saves"`

  // The date this objects was created in RFC 3339
  Created string `json:"created"`

  // The date this object was updated last in RFC 3339
  Updated string `json:"updated"`
}

func NewUserHistory(userId, playlistItemId, mediaId string) *UserHistory {
  return &UserHistory{
    Id:             strings.Replace(uuid.NewUUID().String(), "-", "", -1),
    UserId:         userId,
    PlaylistItemId: playlistItemId,
    MediaId:        mediaId,
    Woots:          0,
    Mehs:           0,
    Saves:          0,
    Created:        time.Now().Format(time.RFC3339),
    Updated:        time.Now().Format(time.RFC3339),
  }
}
