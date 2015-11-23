package db

import (
	"fmt"
	"gopkg.in/mgo.v2/bson"
	"hybris/structs"
	"time"
)

type CommunityHistory struct {
	// Community History Id
	Id bson.ObjectId `json:"id" bson:"_id"`

	// Community Id
	// See /db/community/id
	CommunityId bson.ObjectId `json:"communityId" bson:"communityId"`

	// User Id
	// The user who played the media
	// See /db/user/id
	UserId bson.ObjectId `json:"userId" bson:"userId"`

	// Playlist Item Id
	// The playlist item id
	// See /db/playlistItem/Id
	PlaylistItemId bson.ObjectId `json:"playlistItemId" bson:"playlistItemId"`

	// Global media id
	// See /db/media/id
	MediaId bson.ObjectId `json:"mediaId" bson:"mediaId"`

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

func NewCommunityHistory(communityId, userId, playlistItemId, mediaId bson.ObjectId) *CommunityHistory {
	return &CommunityHistory{
		Id:             bson.NewObjectId(),
		CommunityId:    communityId,
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

func (ch CommunityHistory) Save() error {
	ch.Updated = time.Now()
	_, err := DB.C("communityHistory").UpsertId(ch.Id, ch)
	return err
}

func (ch CommunityHistory) Struct() structs.HistoryItem {
	media, err := GetMedia(bson.M{"mid": ch.MediaId})
	if err != nil {
		fmt.Println(ch.MediaId, err.Error())
		return structs.HistoryItem{}
	}
	mediaInfo := media.Struct()
	return structs.HistoryItem{
		Dj: ch.UserId,
		Media: structs.ResolvedMediaInfo{
			mediaInfo,
			ch.Artist,
			ch.Title,
		},
		Votes: structs.VoteCount{
			Woot: ch.Woots,
			Meh:  ch.Mehs,
			Save: ch.Saves,
		},
	}
}

func StructCommunityHistory(ch []CommunityHistory) []structs.HistoryItem {
	var payload []structs.HistoryItem
	for _, h := range ch {
		payload = append(payload, (&h).Struct())
	}
	return payload
}
