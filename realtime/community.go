package realtime

import (
	"hybris/db/dbcommunity"
	"hybris/db/dbcommunityhistory"
	"hybris/db/dbcommunitystaff"
	"hybris/db/dbmedia"
	"hybris/db/dbplaylist"
	"hybris/db/dbuser"
	"hybris/db/dbuserhistory"
	"hybris/debug"
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
	debug.Log("Creating new realtime community %s", id)
	if c, ok := Communities[id]; ok {
		debug.Log("Realtime community %s already exists", id)
		return c
	}
	c := &Community{
		Id:         id,
		Population: []bson.ObjectId{},
		Waitlist:   []bson.ObjectId{},
		Timer:      time.NewTimer(0),
	}
	debug.Log("Created new realtime community %s", id)
	Communities[id] = c
	return c
}

func (c *Community) Advance() {
	debug.Log("Advancing community %s", c.Id)
	c.Lock()
	defer c.Unlock()

	_ = c.Timer.Stop()

	communityData, err := dbcommunity.GetId(c.Id)
	if err != nil {
		debug.Log("Failed to retrieve community data during advance. Panicking: %s", err.Error())
		c.Panic()
		return
	}

	if c.Media != nil {
		media, err := dbmedia.LockGet(c.Media.Media.Id)
		if err != nil {
			debug.Log("Failed to retrieve media during advance. Panicking: %s", err.Error())
			dbmedia.Unlock(c.Media.Media.Id)
			c.Panic()
			return
		}

		media.Woots += len(c.Media.Votes.Woot)
		media.Mehs += len(c.Media.Votes.Meh)
		media.Saves += len(c.Media.Votes.Save)

		if err := media.Save(); err != nil {
			debug.Log("Failed to save media during advance. Panicking: %s", err.Error())
			dbmedia.Unlock(media.Id)
			c.Panic()
			return
		}

		dbmedia.Unlock(media.Id)

		ch, err := dbcommunityhistory.New(c.Id, c.Media.DjId, c.Media.Media.Id)
		if err != nil {
			debug.Log("Failed to create community history during advance. Panicking: %s", err.Error())
			c.Panic()
			return
		}

		if err := ch.Save(); err != nil {
			debug.Log("Failed to create community history during advance. Panicking: %s", err.Error())
			c.Panic()
			return

		}

		uh, err := dbuserhistory.New(c.Id, c.Media.DjId, c.Media.Media.Id)
		if err != nil {
			debug.Log("Failed to create user history during advance. Panicking: %s", err.Error())
			c.Panic()
			return
		}

		if err := uh.Save(); err != nil {
			debug.Log("Failed to save user history during advance. Panicking: %s", err.Error())
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
			debug.Log("User %s does not exist. Cannot confirm community state. Panicking", userId)
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
			debug.Log("Failed to retrieve playlist %s during advance. Panicking: %s", err.Error())
			c.Panic()
			return
		}

		items, err := playlist.GetItems()

		if err != nil {
			debug.Log("Failed to retrieve playlist items for %s during advance. Panicking: %s", err.Error())
			c.Panic()
			return
		}

		playlistItem := items[0]

		items = append(items[1:], playlistItem)

		if err := playlist.SaveItems(items); err != nil {
			debug.Log("Failed to save playlist items during advance. Panicking: %s", err.Error())
			c.Panic()
			return
		}

		media, err := dbmedia.GetId(playlistItem.MediaId)

		if err != nil {
			debug.Log("Failed to retrieve media %s during advance. Panicking: %s", playlistItem.MediaId, err.Error())
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

	debug.Log("Finished advancing community %s", c.Id)
	// go c.Emit(message.NewEvent("waitlist.update", c.Waitlist))
	// go c.Emit(message.NewEvent("advance", c.Waitlist))
	// Make this just one update
}

func (c *Community) Panic() {
	debug.Log("Community %s is panicking", c.Id)
	for _, v := range c.Population {
		u, ok := Users[v]
		if ok {
			u.Panic()
		}
	}

	debug.Log("Community %s destroyed", c.Id)

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

func (c *Community) Emit(e message.Message) {
	population := c.Population
	for _, p := range population {
		if u, ok := Users[p]; ok {
			go e.Dispatch(u.Client)
		} else {
			debug.Log("User %s in community %s population doesn't exist. Should panic",
				p, c.Id)
		}
	}
}

func (c *Community) Join(id bson.ObjectId) {
	debug.Log("Adding user %s to community %s population", id, c.Id)
	c.Lock()
	defer c.Unlock()

	for _, p := range c.Population {
		if p == id {
			debug.Log("User %s is already in community %s", id, c.Id)
			return
		}
	}

	c.Population = append(c.Population, id)
	debug.Log("Successfully added user %s to community %s population", id, c.Id)
}

func (c *Community) Leave(id bson.ObjectId) {
	debug.Log("Removing user %s from community %s population", id, c.Id)
	c.Lock()
	defer c.Unlock()

	for i, p := range c.Population {
		if p == id {
			c.Population = append(c.Population[:i], c.Population[i+1:]...)
			debug.Log("Successfully removed user %s from community %s population",
				id, c.Id)
		}
	}
	debug.Log("Could not remove user %s from community %s population. Isn't in community",
		id, c.Id)
}

func (c *Community) HasPermission(userId bson.ObjectId, required int) bool {
	debug.Log("Checking to see if user %s has permission %d in community %s",
		userId, required, c.Id)
	staff, err := dbcommunitystaff.GetMulti(-1, uppdb.Cond{"communityId": c.Id})
	if err != nil {
		debug.Log("Could not retrieve community %s staff to check permissions", c.Id)
		return false
	}

	user, _ := dbuser.GetId(userId)

	if user.GlobalRole >= enums.GlobalRoles.TrustedAmbassador {
		debug.Log("User %s in community %s is a trusted ambassador or above. Has all permissions",
			userId, c.Id)
		return true
	}

	if user.GlobalRole >= enums.GlobalRoles.TrialAmbassador && required <= enums.ModerationRoles.Manager {
		debug.Log("User %s in community %s is an ambassador or above. Has manager permissions",
			userId, c.Id)
		return true
	}

	for _, s := range staff {
		if s.UserId == userId {
			debug.Log("User %s in community %s has the required role of %d? %b",
				userId, c.Id, required, s.Role >= required)
			return s.Role >= required
		}
	}

	debug.Log("User %s in community %s is not a staff member")
	return false
}
