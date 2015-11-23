package realtime

import (
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
	if u, ok := Users[id]; ok {
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
	Users[id] = u
	return u
}

func (u User) GetCommunity() *Community {
	return Communities[u.CommunityId]
}

func (u *User) Panic() {
	u.Client.Terminate()
	u.Destroy()
}

func (u *User) Destroy() {
	if community := u.GetCommunity(); community != nil {
		community.Leave(u.Id)
	}
	delete(Users, u.Id)
	u = nil
	// Leave community
}

func (u *User) Hijack(c Client) *User {
	u.Client.Terminate()
	u.Client = c
	return u
}
