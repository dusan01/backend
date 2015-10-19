package db

import (
  "code.google.com/p/go-uuid/uuid"
  "errors"
  "gopkg.in/mgo.v2"
  "gopkg.in/mgo.v2/bson"
  "hybris/downloader"
  "hybris/structs"
  "strings"
  "sync"
  "time"
)

type Media struct {
  sync.Mutex

  // Media Id
  Id string `json:"id" bson:"id"`

  // Type
  // The tpye of media
  // See enum/MEDIA_TYPES
  Type int `json:"type" bson:"type"`

  // Media Id
  // The media Id
  MediaId string `json:"mid" bson:"mid"`

  // Media Image
  // The media image URL
  Image string `json:"img" bson:"img"`

  // Length
  // The length of the media
  Length int `json:"length" bson:"length"`

  // Title
  // The media title
  Title string `json:"title" bson:"title"`

  // Artist
  // The media artist
  Artist string `json:"artist" bson:"artist"`

  // Blurb
  // Short description of the media
  // Has to be 0-200 characters
  //  If it's above 200 characters then trim it to 197 and append an ellipsis
  Blurb string `json:"blurb" bson:"blurb"`

  // Ammount of times this media has been played
  Plays int `json:"plays" bson:"plays"`

  // Ammount of times people have wooted this
  Woots int `json:"woots" bson:"woots"`

  // Amount of times people have meh'd this
  Mehs int `json:"mehs" bson:"mehs"`

  // Amount of times people have saved this
  Saves int `json:"grabs" bson:"grabs"`

  // Playlists
  // The ammount of playlists this media has been inserted into
  Playlists int `json:"playlists" bson:"playlists"`

  // The date this objects was created in RFC 3339
  Created string `json:"created" bson:"created"`

  // The date this object was updated last in RFC 3339
  Updated string `json:"updated" bson:"updated"`
}

func NewMedia(id string, platform int) (*Media, error) {
  if media, err := GetMedia(bson.M{"mediaId": id, "type": platform}); err != mgo.ErrNotFound {
    return media, err
  }

  var (
    image  string
    artist string
    title  string
    blurb  string
    length int
    err    error
  )
  switch platform {
  case 0:
    image, artist, title, blurb, length, err = downloader.Youtube(id)
  case 1:
    image, artist, title, blurb, length, err = downloader.Soundcloud(id)
  default:
    err = errors.New("Invalid type")
  }

  return &Media{
    Id:      strings.Replace(uuid.NewUUID().String(), "-", "", -1),
    Type:    platform,
    MediaId: id,
    Image:   image,
    Artist:  artist,
    Title:   title,
    Blurb:   blurb,
    Length:  length,
    Created: time.Now().Format(time.RFC3339),
    Updated: time.Now().Format(time.RFC3339),
  }, err
}

func GetMedia(query interface{}) (*Media, error) {
  var m Media
  err := DB.C("media").Find(query).One(&m)
  return &m, err
}

func (m Media) Save() error {
  err := DB.C("media").Update(bson.M{"id": m.Id}, m)
  if err == mgo.ErrNotFound {
    return DB.C("media").Insert(m)
  }
  return err
}

func (m Media) Struct() structs.MediaInfo {
  return structs.MediaInfo{
    Id:        m.Id,
    Type:      m.Type,
    MediaId:   m.MediaId,
    Image:     m.Image,
    Length:    m.Length,
    Title:     m.Title,
    Artist:    m.Artist,
    Blurb:     m.Blurb,
    Plays:     m.Plays,
    Woots:     m.Woots,
    Mehs:      m.Mehs,
    Saves:     m.Saves,
    Playlists: m.Playlists,
  }
}
