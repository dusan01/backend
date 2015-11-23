package dbuser

import (
	"errors"
	"hybris/db"
	"hybris/db/dbplaylist"
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
	coll, err := db.Session.Collection("users")
	if err != nil && err != uppdb.ErrCollectionDoesNotExists {
		panic(err)
	}
	collection = coll
}

type User struct {
	// Database object id
	Id bson.ObjectId `json:"id" bson:"_id"`

	// User's username used for URLs and mentions
	Username string `json:"username" bson:"username"`

	// User's display name to display on profiles and in chat
	DisplayName string `json:"displayName" bson:"displayName"`

	// User's email address
	Email string `json:"email" bson:"email"`

	// User's hashed password
	Password []byte `json:"password" bson:"password"`

	// User's global role
	// See enums/GlobalRoles
	GlobalRole int `json:"global_role" bson:"global_role"`

	// Amount of points the user has
	Points int `json:"points" bson:"points"`

	// Facebook user ID used for facebook logins
	FacebookId string `json:"facebookId" bson:"facebookId"`

	// Twitter user ID used for twitter logins
	TwitterId string `json:"twitterId" bson:"twitterId"`

	// Amount of diamonds a user has
	Diamonds int `json:"diamonds" bson:"diamonds"`

	// User's preferred language
	Locale string `json:"locale" bson:"locale"`

	// When the object was created
	Created time.Time `json:"created" bson:"created"`

	// When the object was last updated
	Updated time.Time `json:"updated" bson:"updated"`
}

func New(username string) (User, error) {
	displayName := username
	username = strings.ToLower(username)

	if !validation.Username(username) {
		return User{}, errors.New("invalid username")
	}

	if _, err := get(uppdb.Cond{"username": username}); err == nil {
		return User{}, errors.New("username taken")
	}

	return User{
		Id:          bson.NewObjectId(),
		Username:    username,
		DisplayName: displayName,
		GlobalRole:  2,
		Locale:      "en",
		Created:     time.Now(),
		Updated:     time.Now(),
	}, nil
}

func Get(query interface{}) (User, error) {
	u, err := get(query)
	if u == nil {
		return User{}, err
	}
	return *u, err
}

func get(query interface{}) (*User, error) {
	var user *User
	if err := collection.Find(query).One(&user); err != nil {
		return nil, err
	}
	return getId(user.Id)
}

func GetId(id bson.ObjectId) (User, error) {
	u, err := getId(id)
	if u == nil {
		return User{}, err
	}
	return *u, err
}

func getId(id bson.ObjectId) (*User, error) {
	if _, ok := getMutexes[id]; !ok {
		getMutexes[id] = &sync.Mutex{}
	}

	getMutexes[id].Lock()
	defer getMutexes[id].Unlock()

	if user, found := cache.Get(string(id)); found {
		return user.(*User), nil
	}

	var user *User

	if err := collection.Find(uppdb.Cond{"_id": id}).One(&user); err != nil {
		return nil, err
	}

	cache.Set(string(id), user, gocache.DefaultExpiration)

	return user, nil
}

func GetMulti(max int, query interface{}) (users []User, err error) {
	q := collection.Find(query)
	if max < 0 {
		err = q.All(&users)
	} else {
		err = q.Limit(uint(max)).All(&users)
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

func LockGet(id bson.ObjectId) (*User, error) {
	Lock(id)
	return getId(id)
}

func (u User) Save() (err error) {
	u.Updated = time.Now()
	_, err = collection.Append(u)
	return
}

func (u User) Delete() (err error) {
	cache.Delete(string(u.Id))
	err = collection.Find(uppdb.Cond{"_id": u.Id}).Remove()
	return
}

func (u User) Struct() structs.UserInfo {
	return structs.UserInfo{
		Id:          u.Id,
		Username:    u.Username,
		DisplayName: u.DisplayName,
		GlobalRole:  u.GlobalRole,
		Points:      u.Points,
		Created:     u.Created,
		Updated:     u.Updated,
	}
}

func (u User) PrivateStruct() structs.UserPrivateInfo {
	return structs.UserPrivateInfo{
		u.Struct(),
		u.Diamonds,
		u.Email,
		u.FacebookId,
		u.TwitterId,
	}
}

func StructMulti(users []User) (payload []structs.UserInfo) {
	for _, u := range users {
		payload = append(payload, u.Struct())
	}
	return
}

// Extra methods

func (u User) GetPlaylists() (playlists []dbplaylist.Playlist, err error) {
	playlists, err = dbplaylist.GetMulti(-1, uppdb.Cond{"ownerId": u.Id})
	playlists = u.sorPlaylists(playlists)
	return
}

func (u User) SavePlaylists(playlists []dbplaylist.Playlist) error {
	for _, playlist := range playlists {
		if err := playlist.Save(); err != nil {
			return err
		}
	}
	return nil
}

func (u User) sorPlaylists(playlists []dbplaylist.Playlist) []dbplaylist.Playlist {
	payload := make([]dbplaylist.Playlist, len(playlists))
	for _, playlist := range playlists {
		payload[playlist.Order] = playlist
	}
	return payload
}

func (u User) recalculateItems(playlists []dbplaylist.Playlist) (payload []dbplaylist.Playlist) {
	for i, playlist := range playlists {
		playlist.Order = i
		payload = append(payload, playlist)
	}
	return
}
