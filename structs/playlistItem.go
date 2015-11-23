package structs

import "gopkg.in/mgo.v2/bson"

type PlaylistItem struct {
	Id         bson.ObjectId `json:"id"`
	PlaylistId bson.ObjectId `json:"playlistId"`
	Title      string        `json:"title"`
	Artist     string        `json:"artist"`
	Order      int           `json:"order"`
	Media      MediaInfo     `json:"media"`
}
