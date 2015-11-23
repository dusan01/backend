package dbcommunitystaff

import (
	"hybris/db"
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
	coll, err := db.Session.Collection("communityStaff")
	if err != nil && err != uppdb.ErrCollectionDoesNotExists {
		panic(err)
	}
	collection = coll
}

type CommunityStaff struct {
	// Database object id
	Id bson.ObjectId `json:"id" bson:"_id"`

	// Community this belongs to
	CommunityId bson.ObjectId `json:"communityId" bson:"communityId"`

	// User this belongs to
	UserId bson.ObjectId `json:"userId" bson:"userId"`

	// The community role
	// See enums/CommunityRoles
	Role int `json:"role" bson:"role"`

	// When the object was created
	Created time.Time `json:"created" bson:"created"`

	// When the object was last updated
	Updated time.Time `json:"updated" bson:"updated"`
}

func New(communityId, userId bson.ObjectId, role int) (CommunityStaff, error) {
	return CommunityStaff{
		Id:          bson.NewObjectId(),
		CommunityId: communityId,
		UserId:      userId,
		Role:        role,
		Created:     time.Now(),
		Updated:     time.Now(),
	}, nil
}

func Get(query interface{}) (CommunityStaff, error) {
	cs, err := get(query)
	if cs == nil {
		return CommunityStaff{}, err
	}
	return *cs, err
}

func get(query interface{}) (*CommunityStaff, error) {
	var communityStaff *CommunityStaff
	if err := collection.Find(query).One(&communityStaff); err != nil {
		return nil, err
	}
	return getId(communityStaff.Id)
}

func GetId(id bson.ObjectId) (CommunityStaff, error) {
	cs, err := getId(id)
	if cs == nil {
		return CommunityStaff{}, err
	}
	return *cs, err
}

func getId(id bson.ObjectId) (*CommunityStaff, error) {
	if _, ok := getMutexes[id]; !ok {
		getMutexes[id] = &sync.Mutex{}
	}

	getMutexes[id].Lock()
	defer getMutexes[id].Unlock()

	if communityStaff, found := cache.Get(string(id)); found {
		return communityStaff.(*CommunityStaff), nil
	}

	var communityStaff *CommunityStaff

	if err := collection.Find(uppdb.Cond{"_id": id}).One(&communityStaff); err != nil {
		return nil, err
	}

	cache.Set(string(id), communityStaff, gocache.DefaultExpiration)

	return communityStaff, nil
}

func GetMulti(max int, query interface{}) (communityStaff []CommunityStaff, err error) {
	q := collection.Find(query)
	if max < 0 {
		err = q.All(&communityStaff)
	} else {
		err = q.Limit(uint(max)).All(&communityStaff)
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

func LockGet(id bson.ObjectId) (*CommunityStaff, error) {
	Lock(id)
	return getId(id)
}

func (cs CommunityStaff) Save() (err error) {
	cs.Updated = time.Now()
	_, err = collection.Append(cs)
	return
}

func (cs CommunityStaff) Delete() error {
	cache.Delete(string(cs.Id))
	return collection.Find(uppdb.Cond{"_id": cs.Id}).Remove()
}

func (cs CommunityStaff) Struct() structs.StaffItem {
	return structs.StaffItem{
		UserId: cs.UserId,
		Role:   cs.Role,
	}
}

func StructMulti(communityStaff []CommunityStaff) (payload []structs.StaffItem) {
	for _, cs := range communityStaff {
		payload = append(payload, cs.Struct())
	}
	return
}
