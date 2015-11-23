package dbplaylistitem

import (
	"hybris/db"
	"hybris/db/dbmedia"
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
	coll, err := db.Session.Collection("playlistitems")
	if err != nil && err != uppdb.ErrCollectionDoesNotExists {
		panic(err)
	}
	collection = coll
}

type PlaylistItem struct {
	// Database object id
	Id bson.ObjectId `json:"id" bson:"_id"`

	// Playlist Id
	// See /db/playlist/id
	PlaylistId bson.ObjectId `json:"playlistId" bson:"playlistId"`

	// Title of the media
	Title string `json:"title" bson:"title"`

	// Artist of the media
	Artist string `json:"artist" bson:"artist"`

	// Media object id
	MediaId bson.ObjectId `json:"mediaId" bson:"mid"`

	// Order of the playlist item
	Order int `json:"order" bson:"order"`

	// When the object was created
	Created time.Time `json:"created" bson:"created"`

	// When the object was last updated
	Updated time.Time `json:"updated" bson:"updated"`
}

func New(playlistId, mediaId bson.ObjectId, title, artist string) (PlaylistItem, error) {
	return PlaylistItem{
		Id:         bson.NewObjectId(),
		PlaylistId: playlistId,
		Title:      title,
		Artist:     artist,
		MediaId:    mediaId,
		Order:      -1,
		Created:    time.Now(),
		Updated:    time.Now(),
	}, nil
}

func Get(query interface{}) (PlaylistItem, error) {
	pi, err := get(query)
	if pi == nil {
		return PlaylistItem{}, err
	}
	return *pi, err
}

func get(query interface{}) (*PlaylistItem, error) {
	var playlistItem *PlaylistItem
	if err := collection.Find(query).One(&playlistItem); err != nil {
		return nil, err
	}
	return getId(playlistItem.Id)
}

func GetId(id bson.ObjectId) (PlaylistItem, error) {
	pi, err := getId(id)
	if pi == nil {
		return PlaylistItem{}, err
	}
	return *pi, err
}

func getId(id bson.ObjectId) (*PlaylistItem, error) {
	if _, ok := getMutexes[id]; !ok {
		getMutexes[id] = &sync.Mutex{}
	}

	getMutexes[id].Lock()
	defer getMutexes[id].Unlock()

	if playlistItem, found := cache.Get(string(id)); found {
		return playlistItem.(*PlaylistItem), nil
	}

	var playlistItem *PlaylistItem

	if err := collection.Find(uppdb.Cond{"_id": id}).One(&playlistItem); err != nil {
		return nil, err
	}

	cache.Set(string(id), playlistItem, gocache.DefaultExpiration)

	return playlistItem, nil
}

func GetMulti(max int, query interface{}) (playlistItems []PlaylistItem, err error) {
	q := collection.Find(query)
	if max < 0 {
		err = q.All(&playlistItems)
	} else {
		err = q.Limit(uint(max)).All(&playlistItems)
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

func LockGet(id bson.ObjectId) (*PlaylistItem, error) {
	Lock(id)
	return getId(id)
}

func (pi PlaylistItem) Save() (err error) {
	pi.Updated = time.Now()
	_, err = collection.Append(pi)
	return
}

func (pi PlaylistItem) Delete() error {
	cache.Delete(string(pi.Id))
	return collection.Find(uppdb.Cond{"_id": pi.Id}).Remove()
}

func (pi PlaylistItem) Struct() structs.PlaylistItem {
	media, err := dbmedia.GetId(pi.MediaId)
	if err != nil {
		return structs.PlaylistItem{}
	}
	return structs.PlaylistItem{
		Id:         pi.Id,
		PlaylistId: pi.PlaylistId,
		Title:      pi.Title,
		Artist:     pi.Artist,
		Order:      pi.Order,
		Media:      media.Struct(),
	}
}

func StructMulti(playlistItem []PlaylistItem) (payload []structs.PlaylistItem) {
	for _, pi := range playlistItem {
		payload = append(payload, pi.Struct())
	}
	return
}
