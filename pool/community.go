package pool

import (
  "gopkg.in/mgo.v2/bson"
  "hybris/db"
  "hybris/debug"
  "hybris/enums"
  "hybris/structs"
  "sync"
  "time"
)

type Community struct {
  sync.Mutex
  C     *db.Community
  M     *structs.CommunityPlayingInfo
  H     *db.CommunityHistory
  P     []*db.User
  W     []string
  Timer *time.Timer
}

var Communities = map[string]*Community{}

func NewCommunity(community *db.Community) *Community {
  if v, ok := Communities[community.Id]; ok {
    return v
  }

  c := &Community{
    C:     community,
    M:     nil,
    H:     nil,
    P:     []*db.User{},
    W:     []string{},
    Timer: time.NewTimer(0),
  }

  Communities[community.Id] = c

  return c
}

func (c *Community) GetState() structs.CommunityState {
  return structs.CommunityState{
    Waitlist:   c.W,
    NowPlaying: c.M,
  }
}

func (c *Community) Advance() {
  c.Lock()
  defer c.Unlock()

  _ = c.Timer.Stop()

  if c.H != nil {
    if err := c.H.Save(); err != nil {
      go debug.Log("[pool > Community.Advance] Failed to save community history. Details: [[[ %s ]]]", c.H.Id)
    }
  }

  if c.M != nil && c.C.DjRecycling {
    c.W = append(c.W, c.M.DjId)
  }

  c.M = nil

  if len(c.W) > 0 {

    playlist, err := Clients[c.W[0]].U.GetActivePlaylist()
    if err != nil {
      go debug.Log("[pool > Community.Advance] Failed to retrieve playlist. Details: [[[ %s ||| %s ]]]", c.W[0], err.Error())
      return
    }

    if playlist == nil {
      go debug.Log("[pool > Community.Advance] Active playlist is nil for user %s", c.W[0])
      return
    }

    items, err := playlist.GetItems()
    if err != nil {
      go debug.Log("[pool > Community.Advance] Failed to get playlist items. Details: [[[ %s ||| %s ]]]", playlist.Id, err.Error())
      return
    }

    playlistItem := items[0]

    items = append(items[1:], playlistItem)

    if err := playlist.SaveItems(items); err != nil {
      go debug.Log("[pool > Community.Advance] Failed to save playlist items. Details: [[[ %s ||| %s ]]]", playlist.Id, err.Error())
      return
    }

    media, err := db.GetMedia(bson.M{"mid": playlistItem.MediaId})
    if err != nil {
      go debug.Log("[pool > Community.Advance] Failed to retrieve media. Details: [[[ %s ||| %s ]]]", playlistItem.MediaId, err.Error())
      return
    }

    c.H = db.NewCommunityHistory(c.C.Id, c.W[0], playlistItem.Id, playlistItem.MediaId)
    c.H.Artist = playlistItem.Artist
    c.H.Title = playlistItem.Title

    c.M = &structs.CommunityPlayingInfo{
      DjId:    c.W[0],
      Started: c.H.Created,
      Media:   structs.ResolvedMediaInfo{media.Struct(), playlistItem.Artist, playlistItem.Title},
      Votes: structs.Votes{
        []string{},
        []string{},
        []string{},
      },
    }

    c.W = c.W[1:]
    c.Timer = time.AfterFunc(time.Duration(media.Length)*time.Second, c.Advance)
  }

  if c.M != nil {
    go c.Emit(NewEvent("waitlist.update", c.W))
    go c.Emit(NewEvent("advance", c.M))
  }
}

func (c *Community) Join(user *db.User) int {
  c.Lock()
  defer c.Unlock()

  for _, p := range c.P {
    if p.Id == user.Id {
      // They're already in the community
      return enums.RESPONSE_CODES.BAD_REQUEST
    }
  }

  // Should look into new solutions. It's likely that due to asynchronous nature, it
  // will send the 'user.join' event to the client that has just jojned the
  // community also
  go c.Emit(NewEvent("user.join", user.Struct()))
  c.P = append(c.P, user)
  return enums.RESPONSE_CODES.OK
}

func (c *Community) Leave(user *db.User) int {
  c.LeaveWaitlist(user)
  c.Lock()
  defer c.Unlock()

  for i, p := range c.P {
    if p.Id == user.Id {
      copy(c.P[i:], c.P[i+1:])
      c.P[len(c.P)-1] = nil
      c.P = c.P[:len(c.P)-1]
      go c.Emit(NewEvent("user.leave", user.Struct()))
      return enums.RESPONSE_CODES.OK
    }
  }

  // They aren't in the community
  return enums.RESPONSE_CODES.BAD_REQUEST
}

func (c *Community) Emit(e Message) {
  population := c.P
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
  if c.M == nil {
    return enums.RESPONSE_CODES.BAD_REQUEST
  }

  if voteType == "save" {
    for _, id := range c.M.Votes.Save {
      if id == user.Id {
        return enums.RESPONSE_CODES.BAD_REQUEST
      }
    }
    c.M.Votes.Save = append(c.M.Votes.Save, user.Id)
    voteType = "woot"
  }

  for i, id := range c.M.Votes.Woot {
    if id == user.Id {
      if voteType == "woot" {
        return enums.RESPONSE_CODES.BAD_REQUEST
      }
      c.M.Votes.Woot = append(c.M.Votes.Woot[:i], c.M.Votes.Woot[i+1:]...)
      break
    }
  }

  for i, id := range c.M.Votes.Meh {
    if id == user.Id {
      if voteType == "meh" {
        return enums.RESPONSE_CODES.BAD_REQUEST
      }
      c.M.Votes.Meh = append(c.M.Votes.Meh[:i], c.M.Votes.Meh[i+1:]...)
      break
    }
  }

  switch voteType {
  case "woot":
    c.M.Votes.Woot = append(c.M.Votes.Woot, user.Id)
  case "meh":
    c.M.Votes.Meh = append(c.M.Votes.Meh, user.Id)
  }

  return enums.RESPONSE_CODES.OK

  // Emit a vote update
}

func (c *Community) Move(id string, position int) int {
  c.Lock()
  defer c.Unlock()

  if position >= len(c.W) {
    // Position is out of bounds
    return enums.RESPONSE_CODES.BAD_REQUEST
  }

  current := -1
  for i, v := range c.W {
    if v == id {
      current = i
      break
    }
  }
  if current < 0 {
    // User isn't in the waitlist
    return enums.RESPONSE_CODES.BAD_REQUEST
  }

  c.W = append(c.W[:current], c.W[current+1:]...)
  c.W = append(c.W[:position], append([]string{id}, c.W[position:]...)...)
  go c.Emit(NewEvent("waitlist.update", c.W))
  return enums.RESPONSE_CODES.OK
}

func (c *Community) JoinWaitlist(user *db.User) int {
  c.Lock()
  defer c.Unlock()

  if len(c.W) > 30 {
    // Waitlist is full
    return enums.RESPONSE_CODES.BAD_REQUEST
  }

  if c.M != nil && c.M.DjId == user.Id {
    // They're currently Djing
    return enums.RESPONSE_CODES.BAD_REQUEST
  }

  for _, id := range c.W {
    if id == user.Id {
      // The user is already in the waitlist
      return enums.RESPONSE_CODES.BAD_REQUEST
    }
  }

  c.W = append(c.W, user.Id)
  if c.M == nil {
    go func() {
      c.Advance()
    }()
  } else {
    go c.Emit(NewEvent("waitlist.update", c.W))
  }
  return enums.RESPONSE_CODES.OK
}

func (c *Community) LeaveWaitlist(user *db.User) int {
  c.Lock()
  defer c.Unlock()

  if c.M != nil && c.M.DjId == user.Id {
    recycling := c.C.DjRecycling
    c.C.DjRecycling = false
    c.Unlock()
    c.Advance()
    c.Lock()
    c.C.DjRecycling = recycling
    return enums.RESPONSE_CODES.OK
  }

  for i, id := range c.W {
    if id == user.Id {
      c.W = append(c.W[:i], c.W[i+1:]...)
      go c.Emit(NewEvent("waitlist.update", c.W))
      return enums.RESPONSE_CODES.OK
    }
  }

  return enums.RESPONSE_CODES.BAD_REQUEST
}

// func (c *Community) SetRole(user *db.User, role int) int {
//   cs := db.NewCommunityStaff()
// }

func (c *Community) GetUser(userId string) *db.User {
  var u *db.User = nil
  for _, v := range c.P {
    if v.Id == userId {
      u = v
      break
    }
  }
  return u
}

func (c *Community) HasPermission(user *db.User, required int) bool {
  staff, err := c.C.GetStaff()
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
