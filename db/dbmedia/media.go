package dbmedia

import (
  "errors"
  "hybris/db"
  "hybris/downloader"
  "hybris/structs"
  "sync"
  "time"

  gocache "github.com/pmylund/go-cache"
  "gopkg.in/mgo.v2/bson"
  uppdb "upper.io/db"
)

var (
  collection  uppdb.Collection
  cache       = gocache.New(db.CacheExpiration, db.CacheCleanupInterval)
  getMutexes  = map[bson.ObjectId]*sync.Mutex{}
  lockMutexes = map[bson.ObjectId]*sync.Mutex{}
)

func init() {
  coll, err := db.Session.Collection("media")
  if err != nil && err != uppdb.ErrCollectionDoesNotExists {
    panic(err)
  }
  collection = coll
}

type Media struct {
  // Database object id
  Id bson.ObjectId `json:"id" bson:"_id"`

  // Type of media
  // See enums/MediaTypes
  Type int `json:"type" bson:"type"`

  // Source media id
  MediaId string `json:"mid" bson:"mid"`

  // Static image for the media
  Image string `json:"img" bson:"img"`

  // The duration of the media
  Length int `json:"length" bson:"length"`

  // Media title, determined by split 1
  Title string `json:"title" bson:"title"`

  // Media artist, determined by split 0
  Artist string `json:"artist" bson:"artist"`

  // Media blurb, max is 300 chars
  Blurb string `json:"blurb" bson:"blurb"`

  // Amount of times the media has been played
  Plays int `json:"plays" bson:"plays"`

  // Amount of times the media has been wooted
  Woots int `json:"woots" bson:"woots"`

  // Amoutn of times the media has been mehd
  Mehs int `json:"mehs" bson:"mehs"`

  // Amount of times the media has been grabbed
  Grabs int `json:"grabs" bson:"grabs"`

  // Amount of times the media has been inserted into a playlist
  Playlists int `json:"playlists" bson:"playlists"`

  // When the object was created
  Created time.Time `json:"created" bson:"created"`

  // When the object was last updated
  Updated time.Time `json:"updated" bson:"updated"`
}

func New(mid string, platform int) (Media, error) {
  if media, err := Get(uppdb.Cond{"mid": mid}); err != uppdb.ErrNoMoreRows {
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
    image, artist, title, blurb, length, err = downloader.Youtube(mid)
  case 1:
    image, artist, title, blurb, length, err = downloader.Soundcloud(mid)
  default:
    err = errors.New("Invalid type.")
  }

  if err != nil {
    return Media{}, err
  }

  return Media{
    Id:      bson.NewObjectId(),
    Type:    platform,
    MediaId: mid,
    Image:   image,
    Artist:  artist,
    Title:   title,
    Blurb:   blurb,
    Length:  length,
    Created: time.Now(),
    Updated: time.Now(),
  }, nil
}

func Get(query interface{}) (Media, error) {
  m, err := get(query)
  if m == nil {
    return Media{}, err
  }
  return *m, err
}

func get(query interface{}) (*Media, error) {
  var media *Media
  if err := collection.Find(query).One(&media); err != nil {
    return nil, err
  }
  return getId(media.Id)
}

func GetId(id bson.ObjectId) (Media, error) {
  m, err := getId(id)
  if m == nil {
    return Media{}, err
  }
  return *m, err
}

func getId(id bson.ObjectId) (*Media, error) {
  if _, ok := getMutexes[id]; !ok {
    getMutexes[id] = &sync.Mutex{}
  }

  getMutexes[id].Lock()
  defer getMutexes[id].Unlock()

  if media, found := cache.Get(string(id)); found {
    return media.(*Media), nil
  }

  var media *Media

  if err := collection.Find(uppdb.Cond{"_id": id}).One(&media); err != nil {
    return nil, err
  }

  cache.Set(string(id), media, gocache.DefaultExpiration)

  return media, nil
}

func GetMulti(max int, query interface{}) (media []Media, err error) {
  q := collection.Find(query)
  if max < 0 {
    err = q.All(&media)
  } else {
    err = q.Limit(uint(max)).All(&media)
  }
  return
}

func Lock(id bson.ObjectId) {
  if _, ok := lockMutexes[id]; !ok {
    lockMutexes[id] = &sync.Mutex{}
  }

  lockMutexes[id].Lock()
}

func Unlock(id bson.ObjectId) {
  if _, ok := lockMutexes[id]; !ok {
    return
  }
  lockMutexes[id].Unlock()
}

func LockGet(id bson.ObjectId) (*Media, error) {
  Lock(id)
  return getId(id)
}

func (m Media) Save() (err error) {
  m.Updated = time.Now()
  _, err = collection.Append(m)
  return
}

func (m Media) Delete() error {
  cache.Delete(string(m.Id))
  return collection.Find(uppdb.Cond{"_id": m.Id}).Remove()
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
    Grabs:     m.Grabs,
    Playlists: m.Playlists,
  }
}

func StructMulti(media []Media) (payload []structs.MediaInfo) {
  for _, m := range media {
    payload = append(payload, m.Struct())
  }
  return
}
