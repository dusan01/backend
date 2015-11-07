package pool

import (
  "gopkg.in/mgo.v2/bson"
  "hybris/db"
  "hybris/debug"
  "hybris/enums"
  "hybris/search"
  "hybris/structs"
  "sync"
  "time"
)

type Community struct {
  sync.Mutex
  Community  *db.Community
  Media      *structs.CommunityPlayingInfo
  History    *db.CommunityHistory
  Population []*db.User
  Waitlist   []bson.ObjectId
  Timer      *time.Timer
}

var Communities = map[bson.ObjectId]*Community{}

func NewCommunity(community *db.Community) *Community {
  if v, ok := Communities[community.Id]; ok {
    return v
  }

  c := &Community{
    Community:  community,
    Media:      nil,
    History:    nil,
    Population: []*db.User{},
    Waitlist:   []bson.ObjectId{},
    Timer:      time.NewTimer(0),
  }

  search.UpsertCommunity(community, 0)
  Communities[community.Id] = c

  return c
}

func GetCommunity(id bson.ObjectId) *Community {
  if v, ok := Communities[id]; ok {
    return v
  }
  return nil
}

func (c *Community) GetLandingInfo() structs.LandingCommunityListing {
  host, err := db.GetUser(bson.M{"_id": c.Community.HostId})
  if err != nil {
    return structs.LandingCommunityListing{}
  }

  media := structs.CommunityPlayingInfo{}
  dj := structs.UserInfo{}
  if c.Media != nil {
    media = *c.Media

    if u := c.GetUser(c.Media.DjId); u != nil {
      dj = u.Struct()
    }
  }

  return structs.LandingCommunityListing{
    Population: len(c.Population),
    Playing:    structs.CommunityFullPlayingInfo{media, dj},
    Info:       structs.CommunityFullInfo{c.Community.Struct(), host.Struct()},
  }
}

func (c *Community) GetState() structs.CommunityState {
  return structs.CommunityState{
    Waitlist:   c.Waitlist,
    NowPlaying: c.Media,
  }
}

func (c *Community) Advance() {
  c.Lock()
  defer c.Unlock()

  _ = c.Timer.Stop()

  if c.History != nil {
    if err := c.History.Save(); err != nil {
      debug.Log("[pool/Community.Advance] Failed to save community history. Reason: %s", err.Error())
    }
  }

  if c.Media != nil && c.Community.DjRecycling {
    c.Waitlist = append(c.Waitlist, c.Media.DjId)
  }

  c.Media = nil

  if len(c.Waitlist) > 0 {

    playlist, err := Clients[c.Waitlist[0]].U.GetActivePlaylist()
    if err != nil {
      debug.Log("[pool/Community.Advance] Failed to retrieve playlist for user %s. Reason: %s", c.Waitlist[0], err.Error())
      return
    }

    if playlist == nil {
      debug.Log("[pool/Community.Advance] Active playlist is nil for user %s", c.Waitlist[0])
      return
    }

    items, err := playlist.GetItems()
    if err != nil {
      debug.Log("[pool/Community.Advance] Failed to get playlist items. Details: [[[ %s ||| %s ]]]", playlist.Id, err.Error())
      return
    }

    playlistItem := items[0]

    items = append(items[1:], playlistItem)

    if err := playlist.SaveItems(items); err != nil {
      debug.Log("[pool/Community.Advance] Failed to save playlist items for palylist %s. Reason: %s", playlist.Id, err.Error())
      return
    }

    media, err := db.GetMedia(bson.M{"mid": playlistItem.MediaId})
    if err != nil {
      debug.Log("[pool/Community.Advance] Failed to retrieve media for id %s. Reason: %s", playlistItem.MediaId, err.Error())
      return
    }

    c.History = db.NewCommunityHistory(c.Community.Id, c.Waitlist[0], playlistItem.Id, playlistItem.MediaId)
    c.History.Artist = playlistItem.Artist
    c.History.Title = playlistItem.Title

    c.Media = &structs.CommunityPlayingInfo{
      DjId:    c.Waitlist[0],
      Started: c.History.Created,
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

  if c.Media != nil {
    go c.Emit(NewEvent("waitlist.update", c.Waitlist))
    go c.Emit(NewEvent("advance", c.Media))
  }
}

func (c *Community) Join(user *db.User) int {
  c.Lock()
  defer c.Unlock()

  for _, p := range c.Population {
    if p.Id == user.Id {
      // They're already in the community
      return enums.RESPONSE_CODES.BAD_REQUEST
    }
  }

  // Should look into new solutions. It's likely that due to asynchronous nature, it
  // will send the 'user.join' event to the client that has just joined the
  // community also
  debug.Log("[pool/Community.Join]  User %s %s joined community %s %s",
    user.Username, user.Id.String(), c.Community.Name, c.Community.Id.String())
  go c.Emit(NewEvent("user.join", user.Struct()))
  c.Population = append(c.Population, user)
  search.UpsertCommunity(c.Community, len(c.Population))
  return enums.RESPONSE_CODES.OK
}

func (c *Community) Leave(user *db.User) int {
  c.LeaveWaitlist(user)
  c.Lock()
  defer c.Unlock()

  for i, p := range c.Population {
    if p.Id == user.Id {
      copy(c.Population[i:], c.Population[i+1:])
      c.Population[len(c.Population)-1] = nil
      c.Population = c.Population[:len(c.Population)-1]
      search.UpsertCommunity(c.Community, len(c.Population))
      debug.Log("[pool/Community.Leave] User %s %s left community %s (%s)",
        user.Username, user.Id.String(), c.Community.Name, c.Community.Id.String())
      go c.Emit(NewEvent("user.leave", user.Struct()))
      return enums.RESPONSE_CODES.OK
    }
  }

  // They aren't in the community
  return enums.RESPONSE_CODES.BAD_REQUEST
}

func (c *Community) Emit(e Message) {
  population := c.Population
  for _, p := range population {
    if client, ok := Clients[p.Id]; ok {
      go e.Dispatch(client)
    }
  }
}

// Finish this shit
func (c *Community) Vote(user *db.User, voteType string) int {
  c.Lock()
  defer c.Unlock()
  if c.Media == nil {
    return enums.RESPONSE_CODES.BAD_REQUEST
  }

  if voteType == "save" {
    for _, id := range c.Media.Votes.Save {
      if id == user.Id {
        return enums.RESPONSE_CODES.BAD_REQUEST
      }
    }
    c.Media.Votes.Save = append(c.Media.Votes.Save, user.Id)
    voteType = "woot"
  }

  for i, id := range c.Media.Votes.Woot {
    if id == user.Id {
      if voteType == "woot" {
        return enums.RESPONSE_CODES.BAD_REQUEST
      }
      c.Media.Votes.Woot = append(c.Media.Votes.Woot[:i], c.Media.Votes.Woot[i+1:]...)
      break
    }
  }

  for i, id := range c.Media.Votes.Meh {
    if id == user.Id {
      if voteType == "meh" {
        return enums.RESPONSE_CODES.BAD_REQUEST
      }
      c.Media.Votes.Meh = append(c.Media.Votes.Meh[:i], c.Media.Votes.Meh[i+1:]...)
      break
    }
  }

  switch voteType {
  case "woot":
    c.Media.Votes.Woot = append(c.Media.Votes.Woot, user.Id)
  case "meh":
    c.Media.Votes.Meh = append(c.Media.Votes.Meh, user.Id)
  }

  return enums.RESPONSE_CODES.OK

  // Emit a vote update
}

func (c *Community) Move(id bson.ObjectId, position int) int {
  c.Lock()
  defer c.Unlock()

  if position >= len(c.Waitlist) {
    // Position is out of bounds
    return enums.RESPONSE_CODES.BAD_REQUEST
  }

  current := -1
  for i, v := range c.Waitlist {
    if v == id {
      current = i
      break
    }
  }
  if current < 0 {
    // User isn't in the waitlist
    return enums.RESPONSE_CODES.BAD_REQUEST
  }

  c.Waitlist = append(c.Waitlist[:current], c.Waitlist[current+1:]...)
  c.Waitlist = append(c.Waitlist[:position], append([]bson.ObjectId{id}, c.Waitlist[position:]...)...)
  go c.Emit(NewEvent("waitlist.update", c.Waitlist))
  return enums.RESPONSE_CODES.OK
}

func (c *Community) JoinWaitlist(user *db.User) int {
  c.Lock()
  defer c.Unlock()

  if len(c.Waitlist) > 30 {
    // Waitlist is full
    return enums.RESPONSE_CODES.BAD_REQUEST
  }

  if c.Media != nil && c.Media.DjId == user.Id {
    // They're currently Djing
    return enums.RESPONSE_CODES.BAD_REQUEST
  }

  for _, id := range c.Waitlist {
    if id == user.Id {
      // The user is already in the waitlist
      return enums.RESPONSE_CODES.BAD_REQUEST
    }
  }

  c.Waitlist = append(c.Waitlist, user.Id)
  if c.Media == nil {
    go func() {
      c.Advance()
    }()
  } else {
    go c.Emit(NewEvent("waitlist.update", c.Waitlist))
  }
  return enums.RESPONSE_CODES.OK
}

func (c *Community) LeaveWaitlist(user *db.User) int {
  c.Lock()
  defer c.Unlock()

  if c.Media != nil && c.Media.DjId == user.Id {
    recycling := c.Community.DjRecycling
    c.Community.DjRecycling = false
    c.Unlock()
    c.Advance()
    c.Lock()
    c.Community.DjRecycling = recycling
    return enums.RESPONSE_CODES.OK
  }

  for i, id := range c.Waitlist {
    if id == user.Id {
      c.Waitlist = append(c.Waitlist[:i], c.Waitlist[i+1:]...)
      go c.Emit(NewEvent("waitlist.update", c.Waitlist))
      return enums.RESPONSE_CODES.OK
    }
  }

  return enums.RESPONSE_CODES.BAD_REQUEST
}

// func (c *Community) SetRole(user *db.User, role int) int {
//   cs := db.NewCommunityStaff()
// }

func (c *Community) GetUser(userId bson.ObjectId) *db.User {
  for _, v := range c.Population {
    if v.Id == userId {
      return v
    }
  }
  return nil
}

func (c *Community) HasPermission(user *db.User, required int) bool {
  staff, err := c.Community.GetStaff()
  if err != nil {
    return false
  }

  if user.GlobalRole >= enums.GLOBAL_ROLES.TRIAL_AMBASSADOR {
    return true
  }

  for _, u := range staff {
    if u.UserId == user.Id {
      return u.Role >= required
    }
  }
  return false
}
