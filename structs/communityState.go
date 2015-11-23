package structs

import "gopkg.in/mgo.v2/bson"

type CommunityState struct {
	Waitlist   []bson.ObjectId       `json:"waitlist"`
	NowPlaying *CommunityPlayingInfo `json:"nowPlaying"`
}
