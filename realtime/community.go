package realtime

import (
	"hybris/db/dbcommunity"
	"hybris/db/dbcommunityhistory"
	"hybris/db/dbcommunitystaff"
	"hybris/db/dbmedia"
	"hybris/db/dbplaylist"
	"hybris/db/dbuser"
	"hybris/db/dbuserhistory"
	"hybris/enums"
	"hybris/socket/message"
	"hybris/structs"
	"sync"
	"time"

	"gopkg.in/mgo.v2/bson"
	uppdb "upper.io/db"
)

var Communities = map[bson.ObjectId]*Community{}

type Community struct {
	sync.Mutex
	Id         bson.ObjectId
	Media      *structs.CommunityPlayingInfo
	Population []bson.ObjectId
	Waitlist   []bson.ObjectId
	Timer      *time.Timer
}

func NewCommunity(id bson.ObjectId) *Community {
	if c, ok := Communities[id]; ok {
		return c
	}
	c := &Community{
		Id:         id,
		Population: []bson.ObjectId{},
		Waitlist:   []bson.ObjectId{},
		Timer:      time.NewTimer(0),
	}
	Communities[id] = c
	return c
}

func (c *Community) Advance() {
	c.Lock()
	defer c.Unlock()

	_ = c.Timer.Stop()

	communityData, err := dbcommunity.GetId(c.Id)

	if err != nil {
		c.Panic()
		return
	}

	if c.Media != nil {
		media, err := dbmedia.LockGet(c.Media.Media.Id)
		if err != nil {
			dbmedia.Unlock(c.Media.Media.Id)
			c.Panic()
			return
		}

		media.Woots += len(c.Media.Votes.Woot)
		media.Mehs += len(c.Media.Votes.Meh)
		media.Grabs += len(c.Media.Votes.Grab)

		if err := media.Save(); err != nil {
			c.Panic()
			return
		}

		dbmedia.Unlock(media.Id)

		ch, err := dbcommunityhistory.New(c.Id, c.Media.DjId, c.Media.Media.Id)
		if err != nil {
			c.Panic()
			return
		}

		if err := ch.Save(); err != nil {
			c.Panic()
			return

		}

		uh, err := dbuserhistory.New(c.Id, c.Media.DjId, c.Media.Media.Id)
		if err != nil {
			c.Panic()
			return
		}

		if err := uh.Save(); err != nil {
			c.Panic()
			return

		}

		if communityData.DjRecycling {
			c.Waitlist = append(c.Waitlist, c.Media.DjId)
		}
	}

	c.Media = nil

	if len(c.Waitlist) > 0 {
		userId := c.Waitlist[0]

		user, ok := Users[userId]
		if !ok {
			c.Panic()
			return
		}

		user.Lock()
		defer user.Unlock()

		playlist, err := dbplaylist.Get(uppdb.Cond{
			"ownerId":  userId,
			"selected": true,
		})

		if err != nil {
			c.Panic()
			return
		}

		items, err := playlist.GetItems()

		if err != nil {
			c.Panic()
			return
		}

		playlistItem := items[0]

		items = append(items[1:], playlistItem)

		if err := playlist.SaveItems(items); err != nil {
			c.Panic()
			return
		}

		media, err := dbmedia.GetId(playlistItem.MediaId)

		if err != nil {
			c.Panic()
			return
		}

		c.Media = &structs.CommunityPlayingInfo{
			DjId:    c.Waitlist[0],
			Started: time.Now(),
			Media:   structs.ResolvedMediaInfo{media.Struct(), playlistItem.Artist, playlistItem.Title},
			Votes: structs.Votes{
				[]bson.ObjectId{},
				[]bson.ObjectId{},
				[]bson.ObjectId{},
			},
		}

		c.Waitlist = c.Waitlist[1:]
		c.Timer = time.AfterFunc(time.Duration(media.Length)*time.Second, c.Advance)
	}

	// go c.Emit(message.NewEvent("waitlist.update", c.Waitlist))
	// go c.Emit(message.NewEvent("advance", c.Waitlist))
	// Make this just one update
}

func (c *Community) Panic() {
	for _, v := range c.Population {
		u, ok := Users[v]
		if ok {
			u.Panic()
		}
	}

	c.Timer.Stop()
	c = nil

	// Emit some sort of emergency event
	// Also, think about a new way of handling fatal errors. This
	// doesn't seem like the best way to do it since it DESTROYS the community.
}

func (c Community) GetState() structs.CommunityState {
	return structs.CommunityState{
		Waitlist:   c.Waitlist,
		NowPlaying: c.Media,
	}
}

/// Advance
// Lock the realtime object
// If the media is currently not nil then
//  Update media stats
//  Create and save histroy object for both user and community
//  If recycling enabled then add the dj to the end of the waitlist
// If the waitlist has atleast 1 user in it then
//  Get the realtime user object (?)
//  Get their active playlist
//  Get the first item in the active playlist
//  Store the first item in memory
//  Move the first item to the end of the playlist
//  Save the items
//  Get the media from the playlist
//  Create objects
//  Start a new timer
// If the media is not nil
//  Emite events

// DISCONNECTION
//
// Client
// -> Terminate
//
//
// Realtime User
// -> Destroy

func (c *Community) Emit(e message.Message) {
	population := c.Population
	for _, p := range population {
		if u, ok := Users[p]; ok {
			go e.Dispatch(u.Client)
		}
	}
}

func (c *Community) Join(id bson.ObjectId) {
	c.Lock()
	defer c.Unlock()

	for _, p := range c.Population {
		if p == id {
			return
		}
	}

	c.Population = append(c.Population, id)
}

func (c *Community) Leave(id bson.ObjectId) {
	c.Lock()
	defer c.Unlock()

	for i, p := range c.Population {
		if p == id {
			c.Population = append(c.Population[:i], c.Population[i+1:]...)
		}
	}
}

func (c *Community) HasPermission(userId bson.ObjectId, required int) bool {
	staff, err := dbcommunitystaff.GetMulti(-1, uppdb.Cond{"communityId": c.Id})
	if err != nil {
		return false
	}

	user, _ := dbuser.GetId(userId)

	if user.GlobalRole >= enums.GlobalRoles.TrialAmbassador {
		return true
	}

	for _, s := range staff {
		if s.UserId == userId {
			return s.Role >= required
		}
	}
	return false
}
