package db

import (
  "gopkg.in/mgo.v2/bson"
  "time"
)

type UserHistory struct {
  // User History Id
  Id bson.ObjectId `json:"id" bson:"_id"`

  // user Id
  // The user that this belongs to
  // See /db/user/id
  UserId bson.ObjectId `json:"userId" bson:"userId"`

  // PlaylistItem Id
  // The id of the playlist item
  // See /db/playliatItem/id
  PlaylistItemId bson.ObjectId `json:"playlistItemId" bson:"playlistItemId"`

  // Media Id
  // The media id
  // See /db/media/id
  MediaId string `json:"mediaId" bson:"mediaId"`

  // Title of the media inherited from PlaylistItem
  // See /db/playlistItem/title
  Title string `json:"title" bson:"title"`

  // Artist of the media inherited from PlaylistItem
  // See /db/playlistItem/artist
  Artist string `json:"artist" bson:"artist"`

  // Ammount of times people have wooted this
  Woots int `json:"woots" bson:"woots"`

  // Amount of times people have meh'd this
  Mehs int `json:"mehs" bson:"mehs"`

  // Amount of times people have saved this
  Saves int `json:"saves" bson:"saves"`

  // The date this objects was created
  Created time.Time `json:"created" bson:"created"`

  // The date this object was updated last
  Updated time.Time `json:"updated" bson:"updated"`
}

func NewUserHistory(userId, playlistItemId bson.ObjectId, mediaId string) *UserHistory {
  return &UserHistory{
    Id:             bson.NewObjectId(),
    UserId:         userId,
    PlaylistItemId: playlistItemId,
    MediaId:        mediaId,
    Woots:          0,
    Mehs:           0,
    Saves:          0,
    Created:        time.Now(),
    Updated:        time.Now(),
  }
}
