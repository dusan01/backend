package dbplaylist

import (
	"errors"
	"hybris/db"
	"hybris/db/dbplaylistitem"
	"hybris/structs"
	"hybris/validation"
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
	coll, err := db.Session.Collection("playlists")
	if err != nil && err != uppdb.ErrCollectionDoesNotExists {
		panic(err)
	}
	collection = coll
}

type Playlist struct {
	// Database object id
	Id bson.ObjectId `json:"id" bson:"_id"`

	// Name of the playlist
	Name string `json:"name" bson:"name"`

	// User who owns this
	OwnerId bson.ObjectId `json:"ownerId" bson:"ownerId"`

	// Whether or not the playlist is selected
	// Only one playlist can be selected at a time
	Selected bool `json:"selected" bson:"selected"`

	// The order that playlists are displayed in the UI
	Order int `json:"order" bson:"order"`

	// When the object was created
	Created time.Time `json:"created" bson:"created"`

	// When the object was last updated
	Updated time.Time `json:"updated" bson:"updated"`
}

func New(name string, ownerId bson.ObjectId) (Playlist, error) {
	if !validation.PlaylistName(name) {
		return Playlist{}, errors.New("invalid playlist name")
	}

	return Playlist{
		Id:      bson.NewObjectId(),
		Name:    name,
		OwnerId: ownerId,
		Order:   -1,
		Created: time.Now(),
		Updated: time.Now(),
	}, nil
}

func Get(query interface{}) (Playlist, error) {
	p, err := get(query)
	if p == nil {
		return Playlist{}, err
	}
	return *p, err
}

func get(query interface{}) (*Playlist, error) {
	var playlist *Playlist
	if err := collection.Find(query).One(&playlist); err != nil {
		return nil, err
	}
	return getId(playlist.Id)
}

func GetId(id bson.ObjectId) (Playlist, error) {
	p, err := getId(id)
	if p == nil {
		return Playlist{}, err
	}
	return *p, err
}

func getId(id bson.ObjectId) (*Playlist, error) {
	if _, ok := getMutexes[id]; !ok {
		getMutexes[id] = &sync.Mutex{}
	}

	getMutexes[id].Lock()
	defer getMutexes[id].Unlock()

	if playlist, found := cache.Get(string(id)); found {
		return playlist.(*Playlist), nil
	}

	var playlist *Playlist

	if err := collection.Find(uppdb.Cond{"_id": id}).One(&playlist); err != nil {
		return nil, err
	}

	cache.Set(string(id), playlist, gocache.DefaultExpiration)

	return playlist, nil
}

func GetMulti(max int, query interface{}) (playlists []Playlist, err error) {
	q := collection.Find(query)
	if max < 0 {
		err = q.All(&playlists)
	} else {
		err = q.Limit(uint(max)).All(&playlists)
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

func LockGet(id bson.ObjectId) (*Playlist, error) {
	Lock(id)
	return getId(id)
}

func (p Playlist) Save() (err error) {
	p.Updated = time.Now()
	_, err = collection.Append(p)
	return
}

func (p Playlist) Delete() error {
	cache.Delete(string(p.Id))
	return collection.Find(uppdb.Cond{"_id": p.Id}).Remove()
}

func (p Playlist) Struct() structs.PlaylistInfo {
	items, err := p.GetItems()
	if err != nil {
		return structs.PlaylistInfo{}
	}

	return structs.PlaylistInfo{
		Name:     p.Name,
		Id:       p.Id,
		OwnerId:  p.OwnerId,
		Selected: p.Selected,
		Order:    p.Order,
		Length:   len(items),
	}
}

func StructMulti(playlists []Playlist) (payload []structs.PlaylistInfo) {
	for _, p := range playlists {
		payload = append(payload, p.Struct())
	}
	return
}

// Extra methods

func (p Playlist) GetItems() (items []dbplaylistitem.PlaylistItem, err error) {
	items, err = dbplaylistitem.GetMulti(-1, uppdb.Cond{"playlistId": p.Id})
	items = p.sorItems(items)
	return
}

func (p Playlist) SaveItems(items []dbplaylistitem.PlaylistItem) error {
	for _, item := range items {
		if err := item.Save(); err != nil {
			return err
		}
	}
	return nil
}

func (p Playlist) sorItems(items []dbplaylistitem.PlaylistItem) []dbplaylistitem.PlaylistItem {
	payload := make([]dbplaylistitem.PlaylistItem, len(items))
	for _, item := range items {
		payload[item.Order] = item
	}
	return payload
}

func (p Playlist) recalculateItems(items []dbplaylistitem.PlaylistItem) (payload []dbplaylistitem.PlaylistItem) {
	for i, item := range items {
		item.Order = i
		payload = append(payload, item)
	}
	return
}
