package dbban

import (
	"errors"
	"hybris/db"
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
	coll, err := db.Session.Collection("bans")
	if err != nil && err != uppdb.ErrCollectionDoesNotExists {
		panic(err)
	}
	collection = coll
}

type Ban struct {
	// Database object id
	Id bson.ObjectId `json:"id" bson:"id"`

	// User who was banned
	BanneeId bson.ObjectId `json:"banneeId" bson:"banneeId"`

	// User who created this ban
	BannerId bson.ObjectId `json:"bannerId" bson:"bannerId"`

	// Community this ban belongs to
	CommunityId bson.ObjectId `json:"communityId" bson:"communityId"`

	// Reason for the ban
	Reason string `json:"reason" bson:"reason"`

	// When the ban expires
	Until *time.Time `json:"until" bson:"until"`

	// When the object was created
	Created time.Time `json:"created" bson:"created"`

	// When the object was last updated
	Updated time.Time `json:"updated" bson:"updated"`
}

func New(banneeId, bannerId, communityId bson.ObjectId, reason string, until *time.Time) (Ban, error) {
	if !validation.Reason(reason) {
		return Ban{}, errors.New("Invalid reason.")
	}

	return Ban{
		Id:          bson.NewObjectId(),
		BanneeId:    banneeId,
		BannerId:    bannerId,
		CommunityId: communityId,
		Reason:      reason,
		Until:       until,
		Created:     time.Now(),
		Updated:     time.Now(),
	}, nil
}

func Get(query interface{}) (Ban, error) {
	p, err := get(query)
	if p == nil {
		return Ban{}, err
	}
	return *p, err
}

func get(query interface{}) (*Ban, error) {
	var ban *Ban
	if err := collection.Find(query).One(&ban); err != nil {
		return nil, err
	}
	return getId(ban.Id)
}

func GetId(id bson.ObjectId) (Ban, error) {
	b, err := getId(id)
	if b == nil {
		return Ban{}, err
	}
	return *b, err
}

func getId(id bson.ObjectId) (*Ban, error) {
	if _, ok := getMutexes[id]; !ok {
		getMutexes[id] = &sync.Mutex{}
	}

	getMutexes[id].Lock()
	defer getMutexes[id].Unlock()

	if ban, found := cache.Get(string(id)); found {
		return ban.(*Ban), nil
	}

	var ban *Ban

	if err := collection.Find(uppdb.Cond{"_id": id}).One(&ban); err != nil {
		return nil, err
	}

	cache.Set(string(id), ban, gocache.DefaultExpiration)

	return ban, nil
}

func GetMulti(max int, query interface{}) (bans []Ban, err error) {
	q := collection.Find(query)
	if max < 0 {
		err = q.All(&bans)
	} else {
		err = q.Limit(uint(max)).All(&bans)
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

func LockGet(id bson.ObjectId) (*Ban, error) {
	Lock(id)
	return getId(id)
}

func (b Ban) Save() (err error) {
	b.Updated = time.Now()
	_, err = collection.Append(b)
	return
}

func (b Ban) Delete() error {
	cache.Delete(string(b.Id))
	return collection.Find(uppdb.Cond{"_id": b.Id}).Remove()
}

func (b Ban) Struct() structs.Ban {
	return structs.Ban{
		Banner: b.BannerId,
		Bannee: b.BanneeId,
		Reason: b.Reason,
		Until:  b.Until,
	}
}

func StructMulti(bans []Ban) (payload []structs.Ban) {
	for _, b := range bans {
		payload = append(payload, b.Struct())
	}
	return
}
