package structs

import (
  "gopkg.in/mgo.v2/bson"
)

type PlaylistInfo struct {
  Id       bson.ObjectId `json:"id"`
  Name     string        `json:"name"`
  OwnerId  bson.ObjectId `json:"ownerId"`
  Selected bool          `json:"selected"`
  Order    int           `json:"order"`

  Length int `json:"length"`
}
