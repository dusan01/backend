package dbcommunityhistory

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
	coll, err := db.Session.Collection("communityHistory")
	if err != nil && err != uppdb.ErrCollectionDoesNotExists {
		panic(err)
	}
	collection = coll
}

type CommunityHistory struct {
	// Database object id
	Id bson.ObjectId `json:"id" bson:"_id"`

	// Community this belongs to
	CommunityId bson.ObjectId `json:"communityId" bson:"communityId"`

	// User who was djing
	UserId bson.ObjectId `json:"userId" bson:"userId"`

	// Database media object id
	MediaId bson.ObjectId `json:"mediaId" bson:"mediaId"`

	// Title of the media inherited from PlaylistItem
	Title string `json:"title" bson:"title"`

	// Artist of the media inherited from PlaylistItem
	Artist string `json:"artist" bson:"artist"`

	// Amount of times people wooted
	Woots int `json:"woots" bson:"woots"`

	// Amount of times people meh'd
	Mehs int `json:"mehs" bson:"mehs"`

	// Amount of times people saved
	Saves int `json:"saves" bson:"saves"`

	// When the object was created
	Created time.Time `json:"created" bson:"created"`

	// When the object was last updated
	Updated time.Time `json:"updated" bson:"updated"`
}

func New(communityId, userId, mediaId bson.ObjectId) (CommunityHistory, error) {
	return CommunityHistory{
		Id:          bson.NewObjectId(),
		CommunityId: communityId,
		UserId:      userId,
		MediaId:     mediaId,
		Woots:       0,
		Mehs:        0,
		Grabs:       0,
		Created:     time.Now(),
		Updated:     time.Now(),
	}, nil
}

func Get(query interface{}) (CommunityHistory, error) {
	ch, err := get(query)
	if ch == nil {
		return CommunityHistory{}, err
	}
	return *ch, err
}

func get(query interface{}) (*CommunityHistory, error) {
	var communityHistory *CommunityHistory
	if err := collection.Find(query).One(&communityHistory); err != nil {
		return nil, err
	}
	return getId(communityHistory.Id)
}

func GetId(id bson.ObjectId) (CommunityHistory, error) {
	ch, err := getId(id)
	if ch == nil {
		return CommunityHistory{}, err
	}
	return *ch, err
}

func getId(id bson.ObjectId) (*CommunityHistory, error) {
	if _, ok := getMutexes[id]; !ok {
		getMutexes[id] = &sync.Mutex{}
	}

	getMutexes[id].Lock()
	defer getMutexes[id].Unlock()

	if communityHistory, found := cache.Get(string(id)); found {
		return communityHistory.(*CommunityHistory), nil
	}

	var communityHistory *CommunityHistory

	if err := collection.Find(uppdb.Cond{"_id": id}).One(&communityHistory); err != nil {
		return nil, err
	}

	cache.Set(string(id), communityHistory, gocache.DefaultExpiration)

	return communityHistory, nil
}

func GetMulti(max int, query interface{}) (communityHistory []CommunityHistory, err error) {
	q := collection.Find(query)
	if max < 0 {
		err = q.All(&communityHistory)
	} else {
		err = q.Limit(uint(max)).All(&communityHistory)
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

func LockGet(id bson.ObjectId) (*CommunityHistory, error) {
	Lock(id)
	return getId(id)
}

func (ch CommunityHistory) Save() (err error) {
	ch.Updated = time.Now()
	_, err = collection.Append(ch)
	return
}

func (ch CommunityHistory) Delete() error {
	cache.Delete(string(ch.Id))
	return collection.Find(uppdb.Cond{"_id": ch.Id}).Remove()
}

func (ch CommunityHistory) Struct() structs.HistoryItem {
	media, err := dbmedia.GetId(ch.MediaId)
	if err != nil {
		return structs.HistoryItem{}
	}
	return structs.HistoryItem{
		Dj: ch.UserId,
		Media: structs.ResolvedMediaInfo{
			media.Struct(),
			ch.Artist,
			ch.Title,
		},
		Votes: structs.VoteCount{
			Woot: ch.Woots,
			Meh:  ch.Mehs,
			Grab: ch.Grabs,
		},
	}
}

func StructMulti(communityHistory []CommunityHistory) (payload []structs.HistoryItem) {
	for _, ch := range communityHistory {
		payload = append(payload, ch.Struct())
	}
	return
}
