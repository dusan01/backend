package realtime

import (
	"hybris/debug"
	"sync"

	"gopkg.in/mgo.v2/bson"
)

var Users = map[bson.ObjectId]*User{}

type Client interface {
	Lock()
	Unlock()
	Send([]byte)
	Terminate()
}

type User struct {
	sync.Mutex
	Id     bson.ObjectId
	Client Client
	// Connected   bool
	Status      string
	CommunityId bson.ObjectId
}

func NewUser(id bson.ObjectId, client Client) *User {
	debug.Log("Creating new realtime user %s", id)
	if u, ok := Users[id]; ok {
		debug.Log("Realtime user %s already exists. Hijacking", id)
		return u.Hijack(client)
	}

	u := &User{
		Id:     id,
		Client: client,
		// Connected: true,
		// Still needs to be implemented
		Status:      "",
		CommunityId: "",
	}
	debug.Log("Created new realtime user %s", u.Id)
	Users[id] = u
	return u
}

func (u User) GetCommunity() *Community {
	debug.Log("Retrieving current community for realtime user %s", u.Id)
	return Communities[u.CommunityId]
}

func (u *User) Panic() {
	debug.Log("Realtime user %s panicking", u.Id)
	u.Client.Terminate()
	u.Destroy()
}

func (u *User) Destroy() {
	debug.Log("Destroying realtime user %s", u.Id)
	if community := u.GetCommunity(); community != nil {
		community.Leave(u.Id)
	}
	delete(Users, u.Id)
	u = nil
	debug.Log("Destroyed realtime user %s", u.Id)
}

func (u *User) Hijack(c Client) *User {
	debug.Log("Hijacking realtime user %s", u.Id)
	u.Client.Terminate()
	u.Client = c
	return u
}
