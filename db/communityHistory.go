package db

import (
  "code.google.com/p/go-uuid/uuid"
  "fmt"
  "gopkg.in/mgo.v2/bson"
  "hybris/structs"
  "strings"
  "sync"
  "time"
)

type CommunityHistory struct {
  sync.Mutex

  // Community History Id
  Id string `json:"id"`

  // Community Id
  // See /db/community/id
  CommunityId string `json:"communityId"`

  // User Id
  // The user who played the media
  // See /db/user/id
  UserId string `json:"userId"`

  // Playlist Item Id
  // The playlist item id
  // See /db/playlistItem/Id
  PlaylistItemId string `json:"playlistItemId"`

  // Global media id
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

func NewCommunityHistory(communityId, userId, playlistItemId, mediaId string) *CommunityHistory {
  return &CommunityHistory{
    Id:             strings.Replace(uuid.NewUUID().String(), "-", "", -1),
    CommunityId:    communityId,
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

func StructCommunityHistory(ch []CommunityHistory) []structs.HistoryItem {
  var payload []structs.HistoryItem
  for _, h := range ch {
    payload = append(payload, (&h).Struct())
  }
  return payload
}

func (ch *CommunityHistory) Struct() structs.HistoryItem {
  media, err := GetMedia(bson.M{"mediaid": ch.MediaId})
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
    Votes: structs.Votes{
      Woot: ch.Woots,
      Meh:  ch.Mehs,
      Save: ch.Saves,
    },
  }
}
