package pool

import (
  "encoding/json"
  "github.com/gorilla/websocket"
  "gopkg.in/mgo.v2"
  "gopkg.in/mgo.v2/bson"
  "hybris/db"
  "hybris/debug"
  "hybris/enums"
  "hybris/search"
  "hybris/structs"
  "math"
  "net/http"
  "sync"
  "time"
)

var Maintenance bool = false

type Client struct {
  sync.Mutex
  U         *db.User
  Conn      *websocket.Conn
  ConnM     sync.Mutex
  Community bson.ObjectId
}

var Clients = map[bson.ObjectId]*Client{}

func NewClient(req *http.Request, conn *websocket.Conn) {
  if Maintenance {
    conn.Close()
    return
  }

  cookie, err := req.Cookie("auth")
  if err != nil {
    go debug.Log("[pool > NewClient] Failed to retrieve auth cookie")
    conn.Close()
    return
  }

  session, err := db.GetSession(bson.M{"cookie": cookie.Value})
  if err != nil {
    go debug.Log("[pool > NewClient] Failed to retieve user session with cookie value: [%s]", cookie.Value)
    conn.Close()
    return
  }

  user, err := db.GetUser(bson.M{"_id": session.UserId})
  if err != nil {
    go debug.Log("[pool > NewClient] Failed to find user with session id: [%s]", session.UserId)
    conn.Close()
    return
  }

  if globalBan, err := db.GetGlobalBan(bson.M{"banneeId": user.Id}); err == nil {
    if globalBan.Until == nil || globalBan.Until.After(time.Now()) {
      conn.Close()
      return
    } else {
      if err := globalBan.Delete(); err != nil {
        go debug.Log("[pool > NewClient] Failed to delete global ban: [%s]", globalBan.Id)
        conn.Close()
        return
      }
    }
  } else if err != mgo.ErrNotFound {
    go debug.Log("[pool > NewClient] Failed to retrieve global ban: [%s]", err.Error())
    conn.Close()
    return
  }

  client := &Client{
    U:         user,
    Conn:      conn,
    Community: "",
  }

  if v, ok := Clients[user.Id]; ok {
    go debug.Log("Client already exists. Terminating old client")
    v.Terminate()
  }

  Clients[user.Id] = client

  client.Send([]byte(`{"hello":true}`))
  go client.Listen()
  go debug.Log("Successfully connected client")
}

func (c *Client) Terminate() {
  if community, ok := Communities[c.Community]; ok {
    _ = community.Leave(c.U)
  }
  c.Conn.Close()
  delete(Clients, c.U.Id)
}

func (c *Client) Listen() {
  defer c.Terminate()
  for {
    if _, msg, err := c.Conn.ReadMessage(); err == nil {
      go c.Receive(msg)
    } else {
      return
    }
  }
}

func (c *Client) Send(data []byte) {
  c.ConnM.Lock()
  defer c.ConnM.Unlock()
  c.Conn.SetWriteDeadline(time.Now().Add(55 * time.Second))
  if err := c.Conn.WriteMessage(websocket.TextMessage, data); err != nil {
    c.Terminate()
  }
}

func (c *Client) Receive(msg []byte) {
  var r struct {
    Id     string          `json:"i"`
    Action string          `json:"a"`
    Data   json.RawMessage `json:"d"`
  }

  if err := json.Unmarshal(msg, &r); err != nil {
    go debug.Log("[pool > client.Receive] Client sent bad data")
    return
  }

  switch r.Action {
  /*
     Admin
  */
  case "adm.broadcast":
    var data struct {
      Type    int    `json:"type"`
      Message string `json:"message"`
    }

    if err := json.Unmarshal(r.Data, &data); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    c.Lock()
    defer c.Unlock()

    if c.U.GlobalRole < enums.GLOBAL_ROLES.ADMIN {
      go NewAction(r.Id, enums.RESPONSE_CODES.FORBIDDEN, r.Action, nil).Dispatch(c)
      return
    }

    event := NewEvent("server.broadcast", data)
    for _, client := range Clients {
      if client.U.GlobalRole < enums.GLOBAL_ROLES.ADMIN {
        go event.Dispatch(client)
      }
    }

    NewAction(r.Id, enums.RESPONSE_CODES.OK, r.Action, nil).Dispatch(c)
  case "adm.globalBan":
    var data struct {
      Id       bson.ObjectId `json:"id"`
      Duration int           `json:"duration"`
      Reason   string        `json:"reason"`
    }

    c.Lock()
    defer c.Unlock()

    if err := json.Unmarshal(r.Data, &data); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    if c.U.GlobalRole < enums.GLOBAL_ROLES.ADMIN {
      go NewAction(r.Id, enums.RESPONSE_CODES.FORBIDDEN, r.Action, nil).Dispatch(c)
      return
    }

    user, err := db.GetUser(bson.M{"_id": data.Id})
    if err != nil {
      go NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    globalBan := db.NewGlobalBan(user.Id, c.U.Id, data.Reason, data.Duration)
    if err := globalBan.Save(); err != nil {
      go NewAction(r.Id, enums.RESPONSE_CODES.SERVER_ERROR, r.Action, nil).Dispatch(c)
      return
    }

    if client, ok := Clients[user.Id]; ok {
      if community, ok := Communities[client.Community]; ok {
        go community.Emit(NewEvent("globalBan", bson.M{
          "banner":  c.U.Id,
          "banneee": client.U.Id,
        }))
      }
      go client.Terminate()
    }

    NewAction(r.Id, enums.RESPONSE_CODES.OK, r.Action, nil).Dispatch(c)
  case "adm.maintenance":
    var data struct {
      Start bool `json:"start"`
    }

    if err := json.Unmarshal(r.Data, &data); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    if c.U.GlobalRole < enums.GLOBAL_ROLES.ADMIN {
      go NewAction(r.Id, enums.RESPONSE_CODES.FORBIDDEN, r.Action, nil).Dispatch(c)
      return
    }

    c.Lock()
    defer c.Unlock()

    Maintenance = data.Start

    if Maintenance {
      for _, client := range Clients {
        if client.U.GlobalRole < enums.GLOBAL_ROLES.ADMIN {
          go client.Terminate()
        }
      }
    }

    NewAction(r.Id, enums.RESPONSE_CODES.OK, r.Action, nil).Dispatch(c)
  /*
     Chat
  */
  case "chat.delete":
    var data struct {
      Id bson.ObjectId `json:"id"`
    }

    if err := json.Unmarshal(r.Data, &data); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    c.Lock()
    defer c.Unlock()

    community := GetCommunity(c.Community)

    chat, err := db.GetChat(bson.M{"_id": data.Id})
    if err == mgo.ErrNotFound {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    } else if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.SERVER_ERROR, r.Action, nil).Dispatch(c)
      return
    }

    if chat.UserId != c.U.Id {
      NewAction(r.Id, enums.RESPONSE_CODES.FORBIDDEN, r.Action, nil).Dispatch(c)
      return
    }

    if chat.CommunityId != c.Community || chat.Deleted {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    if err := chat.Delete(); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.SERVER_ERROR, r.Action, nil).Dispatch(c)
      return
    }

    go community.Emit(NewEvent("chat.delete", bson.M{
      "_id":     chat.Id,
      "deleter": c.U.Id,
    }))

    NewAction(r.Id, enums.RESPONSE_CODES.OK, r.Action, nil).Dispatch(c)
  case "chat.send":
    var data struct {
      Me      bool   `json:"me"`
      Message string `json:"message"`
    }

    if err := json.Unmarshal(r.Data, &data); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    community := GetCommunity(c.Community)

    if mute, err := db.GetMute(bson.M{"muteeId": c.U.Id, "communityId": community.Community.Id}); err == nil {
      if mute.Until == nil || mute.Until.After(time.Now()) {
        NewAction(r.Id, enums.RESPONSE_CODES.FORBIDDEN, r.Action, nil).Dispatch(c)
        return
      } else {
        if err := mute.Delete(); err != nil {
          NewAction(r.Id, enums.RESPONSE_CODES.SERVER_ERROR, r.Action, nil).Dispatch(c)
          return
        }
      }
    } else if err != mgo.ErrNotFound {
      NewAction(r.Id, enums.RESPONSE_CODES.SERVER_ERROR, r.Action, nil).Dispatch(c)
      return
    }

    chat := db.NewChat(c.U.Id, community.Community.Id, data.Me, data.Message)
    if err := chat.Save(); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.SERVER_ERROR, r.Action, nil).Dispatch(c)
      return
    }

    go community.Emit(NewEvent("chat.receive", chat.Struct()))

    // Should this line exists?
    NewAction(r.Id, enums.RESPONSE_CODES.OK, r.Action, nil).Dispatch(c)
  /*
     Community
  */
  case "community.create":
    var data struct {
      Url  string `json:"url"`
      Name string `json:"name"`
      Nsfw bool   `json:"nsfw"`
    }

    if err := json.Unmarshal(r.Data, &data); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    c.Lock()
    defer c.Unlock()

    communities, err := c.U.GetCommunities()
    if err != nil || len(communities) >= 3 {
      NewAction(r.Id, enums.RESPONSE_CODES.FORBIDDEN, r.Action, nil).Dispatch(c)
      return
    }

    community, err := db.NewCommunity(c.U.Id, data.Url, data.Name, data.Nsfw)
    if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    if err := community.Save(); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.SERVER_ERROR, r.Action, nil).Dispatch(c)
      return
    }

    staff := db.NewCommunityStaff(community.Id, c.U.Id, enums.MODERATION_ROLES.HOST)
    if err := staff.Save(); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.SERVER_ERROR, r.Action, nil).Dispatch(c)
      return
    }

    _ = NewCommunity(community)

    NewAction(r.Id, enums.RESPONSE_CODES.OK, r.Action, community.Struct()).Dispatch(c)
  case "community.edit":
    var data struct {
      Id              bson.ObjectId `json:"id"`
      Name            *string       `json:"name"`
      Description     *string       `json:"description"`
      WelcomeMessage  *string       `json:"welcomeMessage"`
      WaitlistEnabled *bool         `json:"waitlistEnabled"`
      DjRecycling     *bool         `json:"djRecycling"`
      Nsfw            *bool         `json:"nsfw"`
    }

    if err := json.Unmarshal(r.Data, &data); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    c.Lock()
    defer c.Unlock()

    communityData, err := db.GetCommunity(bson.M{"_id": data.Id})
    if err == mgo.ErrNotFound {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    } else if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.SERVER_ERROR, r.Action, nil).Dispatch(c)
      return
    }

    community := GetCommunity(communityData.Id)

    // Check the user owns this community
    if !community.HasPermission(c.U, enums.MODERATION_ROLES.HOST) {
      NewAction(r.Id, enums.RESPONSE_CODES.FORBIDDEN, r.Action, nil).Dispatch(c)
      return
    }

    if data.Name != nil {
      name := *data.Name
      if length := len(name); length < 2 || length > 30 {
        NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
        return
      }
      community.Community.Name = name
    }

    if data.Nsfw != nil {
      community.Community.Nsfw = *data.Nsfw
    }

    if data.Description != nil {
      description := *data.Description
      if length := len(description); length < 2 || length > 100 {
        NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
        return
      }
      community.Community.Description = description
    }

    if data.WelcomeMessage != nil {
      welcomeMessage := *data.WelcomeMessage
      if length := len(welcomeMessage); length < 2 || length > 300 {
        NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
        return
      }
      community.Community.WelcomeMessage = welcomeMessage
    }

    if data.WaitlistEnabled != nil {
      community.Community.WaitlistEnabled = *data.WaitlistEnabled
    }

    if data.DjRecycling != nil {
      community.Community.DjRecycling = *data.DjRecycling
    }

    // Save community
    if err := community.Community.Save(); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.SERVER_ERROR, r.Action, nil).Dispatch(c)
      return
    }

    NewAction(r.Id, enums.RESPONSE_CODES.OK, r.Action, community.Community.Struct()).Dispatch(c)
  case "community.getHistory":
    var data struct {
      Id bson.ObjectId `json:"id"`
    }

    if err := json.Unmarshal(r.Data, &data); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    communityData, err := db.GetCommunity(bson.M{"_id": data.Id})
    if err == mgo.ErrNotFound {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    } else if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.SERVER_ERROR, r.Action, nil).Dispatch(c)
      return
    }

    community := GetCommunity(communityData.Id)

    history, err := community.Community.GetHistory(50)
    if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.SERVER_ERROR, r.Action, nil).Dispatch(c)
      return
    }

    NewAction(r.Id, enums.RESPONSE_CODES.OK, r.Action, db.StructCommunityHistory(history)).Dispatch(c)
  case "community.getInfo":
    var data struct {
      Id bson.ObjectId `json:"id"`
    }

    if err := json.Unmarshal(r.Data, &data); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    communityData, err := db.GetCommunity(bson.M{"_id": data.Id})
    if err == mgo.ErrNotFound {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    } else if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.SERVER_ERROR, r.Action, nil).Dispatch(c)
      return
    }

    community := GetCommunity(communityData.Id)

    NewAction(r.Id, enums.RESPONSE_CODES.OK, r.Action, community.Community.Struct()).Dispatch(c)
  case "community.getStaff":
    var data struct {
      Id bson.ObjectId `json:"id"`
    }

    if err := json.Unmarshal(r.Data, &data); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    communityData, err := db.GetCommunity(bson.M{"_id": data.Id})
    if err == mgo.ErrNotFound {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    } else if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.SERVER_ERROR, r.Action, nil).Dispatch(c)
      return
    }

    community := GetCommunity(communityData.Id)

    staff, err := community.Community.GetStaff()
    if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.SERVER_ERROR, r.Action, nil).Dispatch(c)
      return
    }

    NewAction(r.Id, enums.RESPONSE_CODES.OK, r.Action, db.StructCommunityStaff(staff)).Dispatch(c)
  case "community.getState":
    var data struct {
      Id bson.ObjectId `json:"id"`
    }

    if err := json.Unmarshal(r.Data, &data); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    communityData, err := db.GetCommunity(bson.M{"_id": data.Id})
    if err == mgo.ErrNotFound {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    } else if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.SERVER_ERROR, r.Action, nil).Dispatch(c)
      return
    }

    community := GetCommunity(communityData.Id)
    state := community.GetState()

    NewAction(r.Id, enums.RESPONSE_CODES.OK, r.Action, state).Dispatch(c)
  case "community.getUsers":
    var data struct {
      Id bson.ObjectId `json:"id"`
    }

    if err := json.Unmarshal(r.Data, &data); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    communityData, err := db.GetCommunity(bson.M{"_id": data.Id})
    if err == mgo.ErrNotFound {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    } else if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.SERVER_ERROR, r.Action, nil).Dispatch(c)
      return
    }

    community := GetCommunity(communityData.Id)
    population := community.Population
    users := []bson.ObjectId{}
    for _, u := range population {
      users = append(users, u.Id)
    }
    NewAction(r.Id, enums.RESPONSE_CODES.OK, r.Action, users).Dispatch(c)
  case "community.join":
    var data struct {
      Url string `json:"url"`
    }

    if err := json.Unmarshal(r.Data, &data); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    c.Lock()
    defer c.Unlock()

    communityData, err := db.GetCommunity(bson.M{"url": data.Url})
    if err == mgo.ErrNotFound {
      NewAction(r.Id, enums.RESPONSE_CODES.NOT_FOUND, r.Action, nil).Dispatch(c)
      return
    } else if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.SERVER_ERROR, r.Action, nil).Dispatch(c)
      return
    }

    community := GetCommunity(communityData.Id)

    if ban, err := db.GetBan(bson.M{"banneeId": c.U.Id, "communityId": communityData.Id}); err == nil {
      if ban.Until == nil || ban.Until.After(time.Now()) {
        NewAction(r.Id, enums.RESPONSE_CODES.FORBIDDEN, r.Action, nil).Dispatch(c)
        return
      } else {
        if err := ban.Delete(); err != nil {
          NewAction(r.Id, enums.RESPONSE_CODES.SERVER_ERROR, r.Action, nil).Dispatch(c)
          return
        }
      }
    } else if err != mgo.ErrNotFound {
      NewAction(r.Id, enums.RESPONSE_CODES.SERVER_ERROR, r.Action, nil).Dispatch(c)
      return
    }

    if user := community.GetUser(c.U.Id); user == nil {
      if currentCommunity, ok := Communities[c.Community]; ok {
        _ = currentCommunity.Leave(c.U)
      }
    }
    c.Community = community.Community.Id
    NewAction(r.Id, community.Join(c.U), r.Action, bson.M{"id": community.Community.Id}).Dispatch(c)
  case "community.search":
    var data struct {
      Query            string `json:"query"`
      Offset           int    `json:"offset"`
      SortByPopulation bool   `json:"sortByPop"`
    }

    if err := json.Unmarshal(r.Data, &data); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    s := time.Now()
    results := search.Communities(data.Query, data.SortByPopulation)
    results = results[:int(math.Min(50, float64(len(results))))]
    results = results[int(math.Min(float64(data.Offset), float64(len(results)))):]
    go debug.Log("[pool > client.Receive] Took %s to search communities for %s", time.Since(s), data.Query)

    payload := make([]structs.LandingCommunityListing, len(results))
    var wg sync.WaitGroup
    var m sync.Mutex
    for i, result := range results {
      wg.Add(1)
      go func(i int, result search.Result) {
        defer wg.Done()
        community := GetCommunity(result.Community.Id)
        info := community.GetLandingInfo()
        m.Lock()
        defer m.Unlock()
        payload[i] = info
      }(i, result)
    }

    wg.Wait()

    NewAction(r.Id, enums.RESPONSE_CODES.OK, r.Action, payload).Dispatch(c)
  case "community.taken":
    var data struct {
      Url string `json:"url"`
    }

    if err := json.Unmarshal(r.Data, &data); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    _, err := db.GetCommunity(bson.M{"url": data.Url})
    NewAction(r.Id, enums.RESPONSE_CODES.OK, r.Action, bson.M{"taken": err == nil})
  /*
     Dj
  */
  case "dj.join":
    c.Lock()
    defer c.Unlock()

    playlist, err := c.U.GetActivePlaylist()
    if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    items, err := playlist.GetItems()
    if err == mgo.ErrNotFound {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    } else if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.SERVER_ERROR, r.Action, nil).Dispatch(c)
      return
    }

    if len(items) <= 0 {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    community := GetCommunity(c.Community)

    NewAction(r.Id, community.JoinWaitlist(c.U), r.Action, nil).Dispatch(c)
  case "dj.leave":
    c.Lock()
    defer c.Unlock()

    community := GetCommunity(c.Community)
    NewAction(r.Id, community.LeaveWaitlist(c.U), r.Action, nil).Dispatch(c)
  case "dj.skip":
    c.Lock()
    defer c.Unlock()

    community := GetCommunity(c.Community)

    if community.Media != nil && community.Media.DjId == c.U.Id {
      community.Advance()
      NewAction(r.Id, enums.RESPONSE_CODES.OK, r.Action, nil).Dispatch(c)
      return
    }
    NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)

  /*
     Media
  */
  case "media.add":
    var data struct {
      Type       int           `json:"type"`
      MediaId    string        `json:"mid"`
      PlaylistId bson.ObjectId `json:"playlistId"`
    }

    if err := json.Unmarshal(r.Data, &data); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    c.Lock()
    defer c.Unlock()

    playlist, err := db.GetPlaylist(bson.M{"_id": data.PlaylistId})
    if err == mgo.ErrNotFound {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    } else if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.SERVER_ERROR, r.Action, nil).Dispatch(c)
      return
    }

    if playlist.OwnerId != c.U.Id {
      NewAction(r.Id, enums.RESPONSE_CODES.FORBIDDEN, r.Action, nil).Dispatch(c)
      return
    }

    playlistItems, err := playlist.GetItems()
    if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.SERVER_ERROR, r.Action, nil).Dispatch(c)
      return
    }

    if len(playlistItems) >= 500 {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    // Add other error reporting
    media, err := db.NewMedia(data.MediaId, data.Type)
    if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.SERVER_ERROR, r.Action, nil).Dispatch(c)
      return
    }

    for _, item := range playlistItems {
      if item.MediaId == media.Id {
        NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
        return
      }
    }

    if err := media.Save(); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.SERVER_ERROR, r.Action, nil).Dispatch(c)
      return
    }

    playlistItem := db.NewPlaylistItem(data.PlaylistId, media.Id, media.Title, media.Artist)

    playlistItems = append([]db.PlaylistItem{playlistItem}, playlistItems...)
    if err := playlist.SaveItems(playlistItems); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.SERVER_ERROR, r.Action, nil).Dispatch(c)
      return
    }

    NewAction(r.Id, enums.RESPONSE_CODES.OK, r.Action, nil).Dispatch(c)
  case "media.import":
    type dataItem struct {
      Type    int    `json:"type"`
      MediaId string `json:"mid"`
    }
    var data struct {
      PlaylistName string     `json:"playlistName"`
      Items        []dataItem `json:"items"`
    }

    if err := json.Unmarshal(r.Data, &data); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    c.Lock()

    playlists, err := c.U.GetPlaylists()
    if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.SERVER_ERROR, r.Action, nil).Dispatch(c)
      return
    }

    playlist, err := db.NewPlaylist(data.PlaylistName, c.U.Id, true)
    if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    if len(playlists) >= 25 {
      NewAction(r.Id, enums.RESPONSE_CODES.FORBIDDEN, r.Action, nil).Dispatch(c)
      return
    }

    if err := playlist.Save(); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.SERVER_ERROR, r.Action, nil).Dispatch(c)
      return
    }

    if err := playlist.Select(c.U); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.SERVER_ERROR, r.Action, nil).Dispatch(c)
      return
    }

    c.Unlock()

    // How it works
    //  Basically, we define a few things first. The amount passed,
    //  the amount failed and a map to indicate what items have been completed.
    //
    //  We then make sure that we only import a max of 500 items and begin.
    //  We loop through everything, if it fails, increment the failed counter.
    //  If it succeeds then increment the passed counter and append the data to the
    //  map we created earlier.
    //
    //  Once all of this is complete, we create a new slice which will take all of the completed items.
    //  We do this by looping through both the values and the keys and appending them to their appropriate
    //  position.

    var (
      m      sync.Mutex
      wg     sync.WaitGroup
      passed = make(map[int]db.PlaylistItem)
      failed int
    )

    total := len(data.Items)

    if total > 500 {
      failed = total - 500
      data.Items = data.Items[:500]
    }

    wg.Add(len(data.Items))

    for i, item := range data.Items {
      go func(i int, item dataItem) {
        defer wg.Done()
        media, err := db.NewMedia(item.MediaId, item.Type)
        if err != nil {
          m.Lock()
          defer m.Unlock()
          failed++
          return
        }

        if err := media.Save(); err != nil {
          m.Lock()
          defer m.Unlock()
          failed++
          return
        }

        playlistItem := db.NewPlaylistItem(playlist.Id, media.Id, media.Title, media.Artist)
        passed[i] = playlistItem
      }(i, item)
    }

    wg.Wait()

    c.Lock()
    defer c.Unlock()

    playlistItems := make([]db.PlaylistItem, len(passed))
    for k, v := range passed {
      playlistItems[k] = v
    }

    if err := playlist.SaveItems(playlistItems); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.SERVER_ERROR, r.Action, nil).Dispatch(c)
      return
    }

    NewAction(r.Id, enums.RESPONSE_CODES.OK, r.Action, bson.M{
      "playlistId": playlist.Id,
      "passed":     len(passed),
      "failed":     failed,
    }).Dispatch(c)
  case "media.search":
    var data struct {
      Type  int    `json:"type"`
      Query string `json:"query"`
    }

    if err := json.Unmarshal(r.Data, &data); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    var (
      results []structs.SearchResult
      err     error
    )

    switch data.Type {
    case 0:
      results, err = search.Youtube(data.Query)
    case 1:
      results, err = search.Soundcloud(data.Query)
    default:
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.SERVER_ERROR, r.Action, nil).Dispatch(c)
      return
    }

    NewAction(r.Id, enums.RESPONSE_CODES.OK, r.Action, results).Dispatch(c)
  /*
     Moderation
  */
  case "moderation.addDj":
    var data struct {
      UserId bson.ObjectId `json:"userId"`
    }

    if err := json.Unmarshal(r.Data, &data); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    c.Lock()
    defer c.Unlock()

    community := GetCommunity(c.Community)

    if !community.HasPermission(c.U, enums.MODERATION_ROLES.BOUNCER) {
      NewAction(r.Id, enums.RESPONSE_CODES.FORBIDDEN, r.Action, nil).Dispatch(c)
      return
    }

    client, ok := Clients[data.UserId]
    if !ok {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    client.Lock()
    if client.Community != c.Community {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    action := NewAction(r.Id, community.Join(client.U), r.Action, nil)
    client.Unlock()

    action.Dispatch(c)
  case "moderation.ban":
    var data struct {
      UserId bson.ObjectId `json:"userId"`
      Reason string        `json:"reason"`
      Length time.Duration `json:"length"`
    }

    if err := json.Unmarshal(r.Data, &data); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    c.Lock()
    defer c.Unlock()

    if data.Length <= 0 || data.Length > 31536000 || len(data.Reason) > 500 {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    community := GetCommunity(c.Community)

    role := enums.MODERATION_ROLES.BOUNCER
    if data.Length > 86400 {
      role = enums.MODERATION_ROLES.MANAGER
    }

    if !community.HasPermission(c.U, role) {
      NewAction(r.Id, enums.RESPONSE_CODES.FORBIDDEN, r.Action, nil).Dispatch(c)
      return
    }

    client, ok := Clients[data.UserId]
    if !ok {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    if ban, err := db.GetBan(bson.M{"banneeId": client.U.Id, "communityId": community.Community.Id}); err == nil {
      if err := ban.Delete(); err != nil {
        NewAction(r.Id, enums.RESPONSE_CODES.SERVER_ERROR, r.Action, nil).Dispatch(c)
        return
      }
    } else if err != mgo.ErrNotFound {
      NewAction(r.Id, enums.RESPONSE_CODES.SERVER_ERROR, r.Action, nil).Dispatch(c)
      return
    }

    client.Lock()
    if client.Community != c.Community {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    cs, err := db.GetCommunityStaff(bson.M{"communityId": community.Community.Id, "userId": client.U.Id})
    if err == mgo.ErrNotFound {
      cs = db.NewCommunityStaff(community.Community.Id, client.U.Id, 0)
    } else if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.SERVER_ERROR, r.Action, nil).Dispatch(c)
      return
    }

    if !community.HasPermission(c.U, cs.Role+1) {
      NewAction(r.Id, enums.RESPONSE_CODES.FORBIDDEN, r.Action, nil).Dispatch(c)
      return
    }

    until := time.Now().Add(data.Length * time.Second)
    ban := db.NewBan(client.U.Id, c.U.Id, community.Community.Id, data.Reason, &until)

    if err := ban.Save(); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.SERVER_ERROR, r.Action, nil).Dispatch(c)
      return
    }

    rCode := community.Leave(client.U)

    event := NewEvent("moderation.ban", struct {
      UserId bson.ObjectId `json:"userId"`
      Reason string        `json:"reason"`
    }{data.UserId, data.Reason})

    go event.Dispatch(client)
    client.Unlock()

    go community.Emit(event)

    NewAction(r.Id, rCode, r.Action, nil).Dispatch(c)
  case "moderation.clearChat":
    c.Lock()
    defer c.Unlock()

    community := GetCommunity(c.Community)

    if !community.HasPermission(c.U, enums.MODERATION_ROLES.MANAGER) {
      NewAction(r.Id, enums.RESPONSE_CODES.FORBIDDEN, r.Action, nil).Dispatch(c)
      return
    }

    go community.Emit(NewEvent("chat.clear", nil))

    NewAction(r.Id, enums.RESPONSE_CODES.OK, r.Action, nil).Dispatch(c)
  case "moderation.deleteChat":
    var data struct {
      Id bson.ObjectId `json:"id"`
    }

    if err := json.Unmarshal(r.Data, &data); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    c.Lock()
    defer c.Unlock()

    community := GetCommunity(c.Community)

    if !community.HasPermission(c.U, enums.MODERATION_ROLES.BOUNCER) {
      NewAction(r.Id, enums.RESPONSE_CODES.FORBIDDEN, r.Action, nil).Dispatch(c)
      return
    }

    chat, err := db.GetChat(bson.M{"_id": data.Id})
    if err == mgo.ErrNotFound {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    } else if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.SERVER_ERROR, r.Action, nil).Dispatch(c)
      return
    }

    if err := chat.Delete(); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.SERVER_ERROR, r.Action, nil).Dispatch(c)
      return
    }

    go community.Emit(NewEvent("chat.delete", bson.M{
      "id":      data.Id,
      "deleter": c.U.Id,
    }))

    NewAction(r.Id, enums.RESPONSE_CODES.OK, r.Action, nil).Dispatch(c)
  case "moderation.forceSkip":
    c.Lock()
    defer c.Unlock()

    community := GetCommunity(c.Community)

    if !community.HasPermission(c.U, enums.MODERATION_ROLES.BOUNCER) {
      NewAction(r.Id, enums.RESPONSE_CODES.FORBIDDEN, r.Action, nil).Dispatch(c)
      return
    }

    community.Advance()

    NewAction(r.Id, enums.RESPONSE_CODES.OK, r.Action, nil).Dispatch(c)
  case "moderation.kick":
    var data struct {
      UserId bson.ObjectId `json:"userId"`
    }

    if err := json.Unmarshal(r.Data, &data); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    c.Lock()
    defer c.Unlock()

    community := GetCommunity(c.Community)

    if !community.HasPermission(c.U, enums.MODERATION_ROLES.BOUNCER) {
      NewAction(r.Id, enums.RESPONSE_CODES.FORBIDDEN, r.Action, nil).Dispatch(c)
      return
    }

    client, ok := Clients[data.UserId]
    if !ok {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    client.Lock()
    if client.Community != c.Community {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    rCode := community.Leave(client.U)

    event := NewEvent("moderation.kick", struct {
      UserId bson.ObjectId `json:"userId"`
    }{data.UserId})
    go event.Dispatch(client)
    client.Unlock()

    go community.Emit(event)

    NewAction(r.Id, rCode, r.Action, nil).Dispatch(c)
  case "moderation.moveDj":
    var data struct {
      UserId   bson.ObjectId `json:"userId"`
      Position int           `json:"position"`
    }

    if err := json.Unmarshal(r.Data, &data); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    c.Lock()
    defer c.Unlock()

    community := GetCommunity(c.Community)

    if !community.HasPermission(c.U, enums.MODERATION_ROLES.MANAGER) {
      NewAction(r.Id, enums.RESPONSE_CODES.FORBIDDEN, r.Action, nil).Dispatch(c)
      return
    }

    user := community.GetUser(data.UserId)
    if user == nil {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    community.Move(data.UserId, data.Position)
    NewAction(r.Id, enums.RESPONSE_CODES.OK, r.Action, nil).Dispatch(c)
  case "moderation.mute":
    var data struct {
      UserId bson.ObjectId `json:"userId"`
      Reason string        `json:"reason"`
      Length time.Duration `json:"length"`
    }

    if err := json.Unmarshal(r.Data, &data); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    c.Lock()
    defer c.Unlock()

    if data.Length <= 0 || data.Length > 31536000 || len(data.Reason) > 500 {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    community := GetCommunity(c.Community)

    role := enums.MODERATION_ROLES.BOUNCER
    if data.Length > 86400 {
      role = enums.MODERATION_ROLES.MANAGER
    }

    if !community.HasPermission(c.U, role) {
      NewAction(r.Id, enums.RESPONSE_CODES.FORBIDDEN, r.Action, nil).Dispatch(c)
      return
    }

    client, ok := Clients[data.UserId]
    if !ok {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    if ban, err := db.GetMute(bson.M{"muteeId": client.U.Id, "communityId": community.Community.Id}); err == nil {
      if err := ban.Delete(); err != nil {
        NewAction(r.Id, enums.RESPONSE_CODES.SERVER_ERROR, r.Action, nil).Dispatch(c)
        return
      }
    } else if err != mgo.ErrNotFound {
      NewAction(r.Id, enums.RESPONSE_CODES.SERVER_ERROR, r.Action, nil).Dispatch(c)
      return
    }

    client.Lock()
    if client.Community != c.Community {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    cs, err := db.GetCommunityStaff(bson.M{"communityId": community.Community.Id, "userId": client.U.Id})
    if err == mgo.ErrNotFound {
      cs = db.NewCommunityStaff(community.Community.Id, client.U.Id, 0)
    } else if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.SERVER_ERROR, r.Action, nil).Dispatch(c)
      return
    }

    if !community.HasPermission(c.U, cs.Role+1) {
      NewAction(r.Id, enums.RESPONSE_CODES.FORBIDDEN, r.Action, nil).Dispatch(c)
      return
    }

    until := time.Now().Add(data.Length * time.Second)
    ban := db.NewMute(client.U.Id, c.U.Id, community.Community.Id, data.Reason, &until)

    if err := ban.Save(); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.SERVER_ERROR, r.Action, nil).Dispatch(c)
      return
    }

    rCode := community.Leave(client.U)

    event := NewEvent("moderation.mute", struct {
      UserId bson.ObjectId `json:"userId"`
      Reason string        `json:"reason"`
    }{data.UserId, data.Reason})

    go event.Dispatch(client)
    client.Unlock()

    go community.Emit(event)

    NewAction(r.Id, rCode, r.Action, nil).Dispatch(c)
  case "moderation.removeDj":
    var data struct {
      UserId bson.ObjectId `json:"userId"`
    }

    if err := json.Unmarshal(r.Data, &data); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    c.Lock()
    defer c.Unlock()

    community := GetCommunity(c.Community)

    if !community.HasPermission(c.U, enums.MODERATION_ROLES.BOUNCER) {
      NewAction(r.Id, enums.RESPONSE_CODES.FORBIDDEN, r.Action, nil).Dispatch(c)
      return
    }

    client, ok := Clients[data.UserId]
    if !ok {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    client.Lock()
    if client.Community != c.Community {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    action := NewAction(r.Id, community.LeaveWaitlist(client.U), r.Action, nil)
    client.Unlock()

    action.Dispatch(c)
  case "moderation.setRole":
    var data struct {
      UserId bson.ObjectId `json:"userId"`
      Role   int           `json:"role"`
    }

    if err := json.Unmarshal(r.Data, &data); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    c.Lock()
    defer c.Unlock()

    community := GetCommunity(c.Community)

    community.Lock()
    defer community.Unlock()

    // Check the role isn't out of biynds
    if data.Role < enums.MODERATION_ROLES.USER || data.Role > enums.MODERATION_ROLES.HOST {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    if !community.HasPermission(c.U, data.Role+1) {
      NewAction(r.Id, enums.RESPONSE_CODES.FORBIDDEN, r.Action, nil).Dispatch(c)
      return
    }

    user := community.GetUser(data.UserId)
    if user == nil {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    cs, err := db.GetCommunityStaff(bson.M{"communityId": c.Community, "userId": data.UserId})
    if err == mgo.ErrNotFound {
      cs = db.NewCommunityStaff(community.Community.Id, user.Id, data.Role)
    } else if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.SERVER_ERROR, r.Action, nil).Dispatch(c)
      return
    }

    if !community.HasPermission(c.U, cs.Role+1) {
      NewAction(r.Id, enums.RESPONSE_CODES.FORBIDDEN, r.Action, nil).Dispatch(c)
      return
    }

    cs.Role = data.Role

    if cs.Role <= enums.MODERATION_ROLES.USER {
      err = cs.Delete()
    } else {
      err = cs.Save()
    }

    if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.SERVER_ERROR, r.Action, nil).Dispatch(c)
      return
    }

    NewAction(r.Id, enums.RESPONSE_CODES.OK, r.Action, nil).Dispatch(c)
  /*
     Playlist
  */
  case "playlist.activate":
    var data struct {
      PlaylistId bson.ObjectId `json:"playlistId"`
    }

    if err := json.Unmarshal(r.Data, &data); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    c.Lock()
    defer c.Unlock()

    playlist, err := db.GetPlaylist(bson.M{"_id": data.PlaylistId})
    if err == mgo.ErrNotFound {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    } else if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.SERVER_ERROR, r.Action, nil).Dispatch(c)
      return
    }

    if playlist.OwnerId != c.U.Id {
      NewAction(r.Id, enums.RESPONSE_CODES.FORBIDDEN, r.Action, nil).Dispatch(c)
      return
    }

    items, err := playlist.GetItems()
    if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.SERVER_ERROR, r.Action, nil).Dispatch(c)
      return
    }

    if len(items) <= 0 {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    if err := playlist.Select(c.U); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.SERVER_ERROR, r.Action, nil).Dispatch(c)
      return
    }

    NewAction(r.Id, enums.RESPONSE_CODES.OK, r.Action, nil).Dispatch(c)
  case "playlist.create":
    var data struct {
      Name string `json:"name"`
    }

    if err := json.Unmarshal(r.Data, &data); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    c.Lock()
    defer c.Unlock()

    playlists, err := c.U.GetPlaylists()
    if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.SERVER_ERROR, r.Action, nil).Dispatch(c)
      return
    }

    playlist, err := db.NewPlaylist(data.Name, c.U.Id, true)
    if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    if len(playlists) >= 25 {
      NewAction(r.Id, enums.RESPONSE_CODES.FORBIDDEN, r.Action, nil).Dispatch(c)
      return
    }

    playlists = append(playlists, *playlist)

    if err := c.U.SavePlaylists(playlists); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.SERVER_ERROR, r.Action, nil).Dispatch(c)
      return
    }

    NewAction(r.Id, enums.RESPONSE_CODES.OK, r.Action, playlist.Struct()).Dispatch(c)
  case "playlist.delete":
    var data struct {
      PlaylistId bson.ObjectId `json:"playlistId"`
    }

    if err := json.Unmarshal(r.Data, &data); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    c.Lock()
    defer c.Unlock()

    playlists, err := c.U.GetPlaylists()
    if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.SERVER_ERROR, r.Action, nil).Dispatch(c)
      return
    }

    playlist, err := db.GetPlaylist(bson.M{"_id": data.PlaylistId})
    if err == mgo.ErrNotFound {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    } else if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.SERVER_ERROR, r.Action, nil).Dispatch(c)
      return
    }

    if playlist.OwnerId != c.U.Id {
      NewAction(r.Id, enums.RESPONSE_CODES.FORBIDDEN, r.Action, nil).Dispatch(c)
      return
    }

    if playlist.Selected {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    for i, p := range playlists {
      if p.Id == data.PlaylistId {
        if err := playlist.Delete(); err != nil {
          NewAction(r.Id, enums.RESPONSE_CODES.SERVER_ERROR, r.Action, nil).Dispatch(c)
          return
        }
        playlists = append(playlists[:i], playlists[i+1:]...)
        break
      }
    }

    if err := c.U.SavePlaylists(playlists); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.SERVER_ERROR, r.Action, nil).Dispatch(c)
      return
    }

    NewAction(r.Id, enums.RESPONSE_CODES.OK, r.Action, nil).Dispatch(c)
  case "playlist.edit":
    var data struct {
      PlaylistId bson.ObjectId `json:"playlistId"`
      Name       *string       `json:"name"`
    }

    if err := json.Unmarshal(r.Data, &data); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    c.Lock()
    defer c.Unlock()

    playlist, err := db.GetPlaylist(bson.M{"_id": data.PlaylistId})
    if err == mgo.ErrNotFound {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    } else if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.SERVER_ERROR, r.Action, nil).Dispatch(c)
      return
    }

    if playlist.OwnerId != c.U.Id {
      NewAction(r.Id, enums.RESPONSE_CODES.FORBIDDEN, r.Action, nil).Dispatch(c)
      return
    }

    if data.Name != nil {
      name := *data.Name
      if length := len(name); length < 1 || length > 30 {
        NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
        return
      }
      playlist.Name = name
    }

    if err := playlist.Save(); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.SERVER_ERROR, r.Action, nil).Dispatch(c)
      return
    }

    NewAction(r.Id, enums.RESPONSE_CODES.OK, r.Action, playlist.Struct()).Dispatch(c)
  case "playlist.get":
    var data struct {
      PlaylistId bson.ObjectId `json:"playlistId"`
    }

    if err := json.Unmarshal(r.Data, &data); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    c.Lock()
    defer c.Unlock()

    playlist, err := db.GetPlaylist(bson.M{"_id": data.PlaylistId})
    if err == mgo.ErrNotFound {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    } else if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.SERVER_ERROR, r.Action, nil).Dispatch(c)
      return
    }

    if playlist.OwnerId != c.U.Id {
      NewAction(r.Id, enums.RESPONSE_CODES.FORBIDDEN, r.Action, nil).Dispatch(c)
      return
    }

    playlistItems, err := playlist.GetItems()
    if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.SERVER_ERROR, r.Action, nil).Dispatch(c)
      return
    }

    NewAction(r.Id, enums.RESPONSE_CODES.OK, r.Action, db.StructPlaylistItems(playlistItems)).Dispatch(c)
  case "playlist.getList":
    c.Lock()
    defer c.Unlock()

    playlists, err := c.U.GetPlaylists()
    if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.SERVER_ERROR, r.Action, nil).Dispatch(c)
      return
    }

    NewAction(r.Id, enums.RESPONSE_CODES.OK, r.Action, db.StructPlaylists(playlists)).Dispatch(c)
  case "playlist.move":
    var data struct {
      PlaylistId bson.ObjectId `json:"playlistId"`
      Position   int           `json:"position"`
    }

    if err := json.Unmarshal(r.Data, &data); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    c.Lock()
    defer c.Unlock()

    playlist, err := db.GetPlaylist(bson.M{"_id": data.PlaylistId})
    if err == mgo.ErrNotFound {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    } else if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.SERVER_ERROR, r.Action, nil).Dispatch(c)
      return
    }

    playlists, err := c.U.GetPlaylists()
    if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.SERVER_ERROR, r.Action, nil).Dispatch(c)
      return
    }

    if playlist.OwnerId != c.U.Id {
      NewAction(r.Id, enums.RESPONSE_CODES.FORBIDDEN, r.Action, nil).Dispatch(c)
      return
    }

    if data.Position < 0 || data.Position > (len(playlists)-1) {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    found := false
    for i, p := range playlists {
      if p.Id == playlist.Id {
        found = true
        playlists = append(playlists[:i], playlists[i+1:]...)
        playlists = append(playlists[:data.Position], append([]db.Playlist{p}, playlists[data.Position:]...)...)
      }
    }

    if !found {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    if err := c.U.SavePlaylists(playlists); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.SERVER_ERROR, r.Action, nil).Dispatch(c)
      return
    }
    NewAction(r.Id, enums.RESPONSE_CODES.OK, r.Action, nil).Dispatch(c)
  /*
     PlaylistItem
  */
  case "playlistItem.delete":
    var data struct {
      PlaylistId     bson.ObjectId `json:"playlistId"`
      PlaylistItemId bson.ObjectId `json:"playlistItemId"`
    }

    if err := json.Unmarshal(r.Data, &data); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    c.Lock()
    defer c.Unlock()

    playlist, err := db.GetPlaylist(bson.M{"_id": data.PlaylistId})
    if err == mgo.ErrNotFound {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    } else if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.SERVER_ERROR, r.Action, nil).Dispatch(c)
      return
    }

    if playlist.OwnerId != c.U.Id {
      NewAction(r.Id, enums.RESPONSE_CODES.FORBIDDEN, r.Action, nil).Dispatch(c)
      return
    }

    playlistItems, err := playlist.GetItems()
    if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.SERVER_ERROR, r.Action, nil).Dispatch(c)
      return
    }

    if len(playlistItems) <= 1 {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    community := GetCommunity(c.Community)

    if community.History != nil && community.History.PlaylistItemId == data.PlaylistItemId {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    for i, pi := range playlistItems {
      if pi.Id == data.PlaylistItemId {
        if err := pi.Delete(); err != nil {
          NewAction(r.Id, enums.RESPONSE_CODES.SERVER_ERROR, r.Action, nil).Dispatch(c)
          return
        }
        playlistItems = append(playlistItems[:i], playlistItems[:i+1]...)
        break
      }
    }

    if err := playlist.SaveItems(playlistItems); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.SERVER_ERROR, r.Action, nil).Dispatch(c)
      return
    }

    NewAction(r.Id, enums.RESPONSE_CODES.OK, r.Action, nil).Dispatch(c)
  case "playlistItem.edit":
    var data struct {
      PlaylistId     bson.ObjectId `json:"playlistId"`
      PlaylistItemId bson.ObjectId `json:"playlistItemId"`
      Artist         *string       `json:"artist"`
      Title          *string       `json:"title"`
    }

    if err := json.Unmarshal(r.Data, &data); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    c.Lock()
    defer c.Unlock()

    playlist, err := db.GetPlaylist(bson.M{"_id": data.PlaylistId})
    if err == mgo.ErrNotFound {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    } else if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.SERVER_ERROR, r.Action, nil).Dispatch(c)
      return
    }

    if playlist.OwnerId != c.U.Id {
      NewAction(r.Id, enums.RESPONSE_CODES.FORBIDDEN, r.Action, nil).Dispatch(c)
      return
    }

    playlistItem, err := db.GetPlaylistItem(bson.M{"playlistId": data.PlaylistId, "id": data.PlaylistItemId})
    if err == mgo.ErrNotFound {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    } else if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.SERVER_ERROR, r.Action, nil).Dispatch(c)
      return
    }

    communityData, err := db.GetCommunity(bson.M{"_id": c.Community})
    if err == mgo.ErrNotFound {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    } else if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.SERVER_ERROR, r.Action, nil).Dispatch(c)
      return
    }

    community := NewCommunity(communityData)
    if community.History != nil && community.History.PlaylistItemId == data.PlaylistItemId {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    if data.Artist != nil {
      artist := *data.Artist
      if len(artist) > 50 {
        NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
        return
      }
      playlistItem.Artist = artist
    }

    if data.Title != nil {
      title := *data.Title
      if len(title) > 50 {
        NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
        return
      }
      playlistItem.Title = title
    }

    if err := playlistItem.Save(); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.SERVER_ERROR, r.Action, nil).Dispatch(c)
      return
    }
    NewAction(r.Id, enums.RESPONSE_CODES.OK, r.Action, nil).Dispatch(c)
  case "playlistItem.move":
    var data struct {
      PlaylistId     bson.ObjectId `json:"playlistId"`
      PlaylistItemId bson.ObjectId `json:"playlistItemId"`
      Position       int           `json:"position"`
    }

    if err := json.Unmarshal(r.Data, &data); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    c.Lock()
    defer c.Unlock()

    playlist, err := db.GetPlaylist(bson.M{"_id": data.PlaylistId})
    if err == mgo.ErrNotFound {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    } else if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.SERVER_ERROR, r.Action, nil).Dispatch(c)
      return
    }

    if playlist.OwnerId != c.U.Id {
      NewAction(r.Id, enums.RESPONSE_CODES.FORBIDDEN, r.Action, nil).Dispatch(c)
      return
    }

    playlistItems, err := playlist.GetItems()
    if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.SERVER_ERROR, r.Action, nil).Dispatch(c)
      return
    }

    if data.Position < 0 || data.Position > (len(playlistItems)-1) {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    found := false
    for i, pi := range playlistItems {
      if pi.Id == data.PlaylistItemId {
        found = true
        playlistItems = append(playlistItems[:i], playlistItems[i+1:]...)
        playlistItems = append(playlistItems[:data.Position], append([]db.PlaylistItem{pi}, playlistItems[data.Position:]...)...)
        break
      }
    }

    if !found {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    if err := playlist.SaveItems(playlistItems); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.SERVER_ERROR, r.Action, nil).Dispatch(c)
      return
    }
    NewAction(r.Id, enums.RESPONSE_CODES.OK, r.Action, nil).Dispatch(c)
  /*
     Vote
  */
  case "vote.woot":
    c.Lock()
    defer c.Unlock()

    community := GetCommunity(c.Community)

    if community.Media != nil && community.Media.DjId == c.U.Id {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    NewAction(r.Id, community.Vote(c.U, "woot"), r.Action, nil).Dispatch(c)
  case "vote.meh":
    c.Lock()
    defer c.Unlock()

    community := GetCommunity(c.Community)
    if community.Media != nil && community.Media.DjId == c.U.Id {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    NewAction(r.Id, community.Vote(c.U, "meh"), r.Action, nil).Dispatch(c)
  case "vote.save":
    var data struct {
      PlaylistId bson.ObjectId `json:"playlistId"`
    }

    if err := json.Unmarshal(r.Data, &data); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    c.Lock()
    defer c.Unlock()

    community := GetCommunity(c.Community)
    if community.Media != nil && community.Media.DjId == c.U.Id {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    // Add the currently playing media to the playlist

    playlist, err := db.GetPlaylist(bson.M{"_id": data.PlaylistId})
    if err == mgo.ErrNotFound {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    } else if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.SERVER_ERROR, r.Action, nil).Dispatch(c)
      return
    }

    if playlist.OwnerId != c.U.Id {
      NewAction(r.Id, enums.RESPONSE_CODES.FORBIDDEN, r.Action, nil).Dispatch(c)
      return
    }

    playlistItems, err := playlist.GetItems()
    if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.SERVER_ERROR, r.Action, nil).Dispatch(c)
      return
    }

    if len(playlistItems) >= 500 {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    // Should add mutex to prevent last second fuck ups

    for _, item := range playlistItems {
      if item.MediaId == community.Media.Media.Id {
        NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
        return
      }
    }

    playlistItem := db.NewPlaylistItem(data.PlaylistId, community.Media.Media.Id, community.Media.Media.Title, community.Media.Media.Artist)

    playlistItems = append([]db.PlaylistItem{playlistItem}, playlistItems...)
    if err := playlist.SaveItems(playlistItems); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.SERVER_ERROR, r.Action, nil).Dispatch(c)
      return
    }

    NewAction(r.Id, community.Vote(c.U, "save"), r.Action, nil).Dispatch(c)
  }
}
