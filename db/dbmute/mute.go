package dbmute

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
	coll, err := db.Session.Collection("mutes")
	if err != nil && err != uppdb.ErrCollectionDoesNotExists {
		panic(err)
	}
	collection = coll
}

type Mute struct {
	// Database object id
	Id bson.ObjectId `json:"id" bson:"_id"`

	// User who got muted
	MuteeId bson.ObjectId `json:"muteeId" bson:"muteeId"`

	// User who created this mute
	MuterId bson.ObjectId `json:"muterId" bson:"muterId"`

	// Community this mute belongs to
	CommunityId bson.ObjectId `json:"communityId" bson:"communityId"`

	// Reason for the mute
	Reason string `json:"reason" bson:"reason"`

	// When the mute expires
	Until *time.Time `json:"until" bson:"until"`

	// When the object was created
	Created time.Time `json:"created" bson:"created"`

	// When the object was last updated
	Updated time.Time `json:"updated" bson:"updated"`
}

func New(muteeId, muterId, communityId bson.ObjectId, reason string, until *time.Time) (Mute, error) {
	if !validation.Reason(reason) {
		return Mute{}, errors.New("Invalid reason.")
	}

	return Mute{
		Id:          bson.NewObjectId(),
		MuteeId:     muteeId,
		MuterId:     muterId,
		CommunityId: communityId,
		Reason:      reason,
		Until:       until,
		Created:     time.Now(),
		Updated:     time.Now(),
	}, nil
}

func Get(query interface{}) (Mute, error) {
	m, err := get(query)
	if m == nil {
		return Mute{}, err
	}
	return *m, err
}

func get(query interface{}) (*Mute, error) {
	var mute *Mute
	if err := collection.Find(query).One(&mute); err != nil {
		return nil, err
	}
	return getId(mute.Id)
}

func GetId(id bson.ObjectId) (Mute, error) {
	m, err := getId(id)
	if m == nil {
		return Mute{}, err
	}
	return *m, err
}

func getId(id bson.ObjectId) (*Mute, error) {
	if _, ok := getMutexes[id]; !ok {
		getMutexes[id] = &sync.Mutex{}
	}

	getMutexes[id].Lock()
	defer getMutexes[id].Unlock()

	if mute, found := cache.Get(string(id)); found {
		return mute.(*Mute), nil
	}

	var mute *Mute

	if err := collection.Find(uppdb.Cond{"_id": id}).One(&mute); err != nil {
		return nil, err
	}

	cache.Set(string(id), mute, gocache.DefaultExpiration)

	return mute, nil
}

func GetMulti(max int, query interface{}) (mutes []Mute, err error) {
	q := collection.Find(query)
	if max < 0 {
		err = q.All(&mutes)
	} else {
		err = q.Limit(uint(max)).All(&mutes)
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

func LockGet(id bson.ObjectId) (*Mute, error) {
	Lock(id)
	return getId(id)
}

func (m Mute) Save() (err error) {
	m.Updated = time.Now()
	_, err = collection.Append(m)
	return
}

func (m Mute) Delete() error {
	cache.Delete(string(m.Id))
	return collection.Find(uppdb.Cond{"_id": m.Id}).Remove()
}

func (m Mute) Struct() structs.Mute {
	return structs.Mute{
		Mutee:  m.MuteeId,
		Muter:  m.MuterId,
		Reason: m.Reason,
		Until:  m.Until,
	}
}

func StructMulti(mutes []Mute) (payload []structs.Mute) {
	for _, m := range mutes {
		payload = append(payload, m.Struct())
	}
	return
}
