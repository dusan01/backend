package dbcommunity

import (
	"errors"
	"hybris/db"
	"hybris/structs"
	"hybris/validation"
	"strings"
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
	coll, err := db.Session.Collection("communities")
	if err != nil && err != uppdb.ErrCollectionDoesNotExists {
		panic(err)
	}
	collection = coll
}

type Community struct {
	// Database object id
	Id bson.ObjectId `json:"id" bson:"_id"`

	// Community URL
	Url string `json:"url" bson:"url"`

	// Community name
	Name string `json:"name" bson:"name"`

	// User who owns the community
	HostId bson.ObjectId `json:"hostId" bson:"hostId"`

	// Community description
	Description string `json:"description" bson:"description"`

	// Welcome Message
	WelcomeMessage string `json:"welcomeMessage" bson:"welcomeMessage"`

	// Whether or not the waitlist is enabled
	WaitlistEnabled bool `json:"waitlistEnabled" bson:"waitlistEnabled"`

	// Whether or not dj recycling is enabled
	DjRecycling bool `json:"djRecycling" bson:"djRecycling"`

	// Whether or not the community is marked as NSFW
	Nsfw bool `json:"nsfw" bson:"nsfw"`

	// When the object was created
	Created time.Time `json:"created" bson:"created"`

	// When the object was last updated
	Updated time.Time `json:"updated" bson:"updated"`
}

func New(host bson.ObjectId, url, name string, nsfw bool) (Community, error) {
	url = strings.ToLower(url)

	if !validation.CommunityUrl(url) {
		return Community{}, errors.New("Invalid community url.")
	}

	if !validation.CommunityName(name) {
		return Community{}, errors.New("Invalid community name.")
	}

	if _, err := Get(uppdb.Cond{"url": url}); err == nil {
		return Community{}, errors.New("Community already exists.")
	}

	return Community{
		Id:              bson.NewObjectId(),
		Url:             url,
		Name:            name,
		HostId:          host,
		WaitlistEnabled: true,
		DjRecycling:     true,
		Nsfw:            nsfw,
		Created:         time.Now(),
		Updated:         time.Now(),
	}, nil
}

func Get(query interface{}) (Community, error) {
	c, err := get(query)
	if c == nil {
		return Community{}, err
	}
	return *c, err
}

func get(query interface{}) (*Community, error) {
	var community *Community
	err := collection.Find(query).One(&community)
	if err != nil {
		return nil, err
	}

	return getId(community.Id)
}

func GetId(id bson.ObjectId) (Community, error) {
	c, err := getId(id)
	if c == nil {
		return Community{}, err
	}
	return *c, err
}

func getId(id bson.ObjectId) (*Community, error) {
	if _, ok := getMutexes[id]; !ok {
		getMutexes[id] = &sync.Mutex{}
	}

	getMutexes[id].Lock()
	defer getMutexes[id].Unlock()

	if community, found := cache.Get(string(id)); found {
		return community.(*Community), nil
	}

	var community *Community

	if err := collection.Find(uppdb.Cond{"_id": id}).One(&community); err != nil {
		return nil, err
	}

	cache.Set(string(id), community, -1)

	return community, nil
}

func GetMulti(max int, query interface{}) (communities []Community, err error) {
	q := collection.Find(query)
	if max < 0 {
		err = q.All(&communities)
	} else {
		err = q.Limit(uint(max)).All(&communities)
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

func LockGet(id bson.ObjectId) (*Community, error) {
	Lock(id)
	return getId(id)
}

func (c Community) Save() (err error) {
	c.Updated = time.Now()
	_, err = collection.Append(c)
	return
}

func (c Community) Delete() error {
	cache.Delete(string(c.Id))
	return collection.Find(uppdb.Cond{"_id": c.Id}).Remove()
}

func (c Community) Struct() structs.CommunityInfo {
	return structs.CommunityInfo{
		Id:              c.Id,
		Url:             c.Url,
		Name:            c.Name,
		HostId:          c.HostId,
		Description:     c.Description,
		WelcomeMessage:  c.WelcomeMessage,
		WaitlistEnabled: c.WaitlistEnabled,
		DjRecycling:     c.DjRecycling,
		Nsfw:            c.Nsfw,
	}
}

func StructMulti(communities []Community) (payload []structs.CommunityInfo) {
	for _, c := range communities {
		payload = append(payload, c.Struct())
	}
	return
}
