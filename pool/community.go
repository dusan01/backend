package pool

import (
  "fmt"
  "gopkg.in/mgo.v2/bson"
  "hybris/db"
  "sync"
  "time"
)

type Community struct {
  sync.Mutex
  C     *db.Community
  M     *db.CommunityHistory
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
    P:     []*db.User{},
    W:     []string{},
    Timer: time.NewTimer(0),
  }

  Communities[community.Id] = c

  return c
}

func (c *Community) Advance() {
  c.Lock()
  defer c.Unlock()

  _ = c.Timer.Stop()

  if c.M != nil && c.C.DjRecycling {
    c.W = append(c.W, c.M.UserId)
  }

  c.M = nil

  if len(c.W) > 0 {

    playlist, err := Clients[c.W[0]].U.GetActivePlaylist()
    if err != nil {
      fmt.Printf("[FATAL] Failed to retrieve playlist. Details: [[[ %s ||| %s ]]]\n", c.W[0], err.Error())
      return
    }

    if playlist == nil {
      fmt.Printf("[FATAL] Active playlist is nil for user %s\n", c.W[0])
      return
    }

    items, err := playlist.GetItems()
    if err != nil {
      fmt.Printf("[FATAL] Failed to get playlist items. Details: [[[ %s ||| %s ]]]\n", playlist.Id, err.Error())
      return
    }

    playlistItem := items[0]

    items = append(items[1:], playlistItem)

    if err := playlist.SaveItems(items); err != nil {
      fmt.Printf("[FATAL] Failed to save playlist items. Details: [[[ %s ||| %s ]]]\n", playlist.Id, err.Error())
      return
    }

    media, err := db.GetMedia(bson.M{"mediaid": playlistItem.MediaId})
    if err != nil {
      fmt.Printf("[FATAL] Failed to retrieve media. Details: [[[ %s ||| %s ]]]\n", playlistItem.MediaId, err.Error())
      return
    }

    c.M = db.NewCommunityHistory(c.C.Id, c.W[0], playlistItem.Id, playlistItem.MediaId)
    c.M.Artist = playlistItem.Artist
    c.M.Title = playlistItem.Title

    c.W = c.W[1:]
    c.Timer = time.AfterFunc(time.Duration(media.Length)*time.Second, c.Advance)
  }

  go c.Emit(NewEvent(-1, 0, "advance", c.M.Struct()))
}

func (c *Community) Join(user *db.User) {
  c.Lock()
  defer c.Unlock()

  for _, p := range c.P {
    if p.Id == user.Id {
      return
    }
  }

  c.P = append(c.P, user)
  // Emit event
}

func (c *Community) Leave(user *db.User) {
  c.Lock()
  defer c.Unlock()

  c.LeaveWaitlist(user)

  for i, p := range c.P {
    if p.Id == user.Id {
      copy(c.P[i:], c.P[i+1:])
      c.P[len(c.P)-1] = nil
      c.P = c.P[:len(c.P)-1]
      break
    }
  }

  for i, v2 := range c.W {
    if v2 == user.Id {
      c.W = append(c.W[:i], c.W[i+1:]...)
      break
    }
  }

  // Done!
}

func (c *Community) Emit(e *Event) {
  for _, p := range c.P {
    go e.Dispatch(Clients[p.Id])
  }
}

func (c *Community) Vote(voteType int, user *db.User) {
  c.Lock()
  defer c.Unlock()
  if c.M == nil {
    return
  }

}

func (c *Community) JoinWaitlist(user *db.User) {
  c.Lock()
  defer c.Unlock()

  if c.M != nil && c.M.UserId == user.Id {
    // They're currently Djing
    return
  }

  for _, id := range c.W {
    if id == user.Id {
      // The user is already in the waitlist
      return
    }
  }

  c.W = append(c.W, user.Id)
  if c.M == nil {
    go func() {
      c.Advance()
    }()
  }
  // Done! Emit a waitlist update
}

func (c *Community) LeaveWaitlist(user *db.User) {
  c.Lock()
  defer c.Unlock()

  if c.M != nil && c.M.UserId == user.Id {
    recycling := c.C.DjRecycling
    c.C.DjRecycling = false
    c.Unlock()
    c.Advance()
    c.Lock()
    c.C.DjRecycling = recycling
    return
  }

  for i, id := range c.W {
    if id == user.Id {
      c.W = append(c.W[:i], c.W[i+1:]...)
      // Done! Emit a waitlist update
      return
    }
  }

  // The user isn't Djing or in the waitlist, return a BAD_Request (1)
}
