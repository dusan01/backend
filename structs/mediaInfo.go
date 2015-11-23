package structs

import "gopkg.in/mgo.v2/bson"

type MediaInfo struct {
	Id        bson.ObjectId `json:"id"`
	Type      int           `json:"type"`
	MediaId   string        `json:"mid"`
	Image     string        `json:"img"`
	Length    int           `json:"length"`
	Title     string        `json:"title"`
	Artist    string        `json:"artist"`
	Blurb     string        `json:"blurb"`
	Plays     int           `json:"plays"`
	Woots     int           `json:"woots"`
	Mehs      int           `json:"mehs"`
	Grabs     int           `json:"grabs"`
	Playlists int           `json:"playlists"`
}
