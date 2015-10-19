package pool

import (
  "bytes"
  "encoding/json"
  "fmt"
  "github.com/gorilla/websocket"
  "gopkg.in/mgo.v2"
  "gopkg.in/mgo.v2/bson"
  "hybris/db"
  "hybris/enums"
  "hybris/searcher"
  "hybris/structs"
  "net/http"
  "sync"
  "time"
)

var Maintenance bool = false

type Client struct {
  sync.Mutex
  U         *db.User
  Conn      *websocket.Conn
  Community string
}

var Clients = map[string]*Client{}

func NewClient(req *http.Request, conn *websocket.Conn) {
  if Maintenance {
    conn.Close()
    return
  }

  cookie, err := req.Cookie("auth")
  if err != nil {
    conn.Close()
    return
  }

  session, err := db.GetSession(bson.M{"cookie": cookie.Value})
  if err != nil {
    conn.Close()
    return
  }

  user, err := db.GetUser(bson.M{"id": session.UserId})
  if err != nil {
    conn.Close()
    return
  }

  client := &Client{
    U:         user,
    Conn:      conn,
    Community: "",
  }

  if v, ok := Clients[user.Id]; ok {
    v.Terminate()
  }

  Clients[user.Id] = client

  client.Send([]byte(`{"hello":true}`))
  go client.Listen()
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
    var (
      msg []byte
      err error
    )

    if _, msg, err = c.Conn.ReadMessage(); err != nil {
      return
    }

    go c.Receive(msg)
  }
}

func (c *Client) Send(data []byte) {
  c.Conn.SetWriteDeadline(time.Now().Add(55 * time.Second))
  if err := c.Conn.WriteMessage(websocket.TextMessage, data); err != nil {
    fmt.Printf("[FATAL] Failed to send websocket message [[[ %s ]]]\n", err.Error())
    c.Terminate()
  }
}

func (c *Client) Receive(msg []byte) {
  var r struct {
    Id     int             `json:"i"`
    Action string          `json:"a"`
    Data   json.RawMessage `json:"d"`
  }

  decoder := json.NewDecoder(bytes.NewReader(msg))
  if err := decoder.Decode(&r); err != nil {
    return
  }

  switch r.Action {
  /*
     Admin
  */
  case "adm.broadcast":
    c.Lock()
    defer c.Unlock()

    var data struct {
      Type    int    `json:"type"`
      Message string `json:"message"`
    }

    if err := json.Unmarshal(r.Data, &data); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    if c.U.GlobalRole < enums.GLOBAL_ROLES.ADMIN {
      go NewAction(r.Id, enums.RESPONSE_CODES.UNAUTHORIZED, r.Action, nil).Dispatch(c)
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
    c.Lock()
    defer c.Unlock()

    var data struct {
      Id       string `json:"id"`
      Duration int    `json:"duration"`
      Reason   string `json:"reason"`
    }

    if err := json.Unmarshal(r.Data, &data); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    if c.U.GlobalRole < enums.GLOBAL_ROLES.ADMIN {
      go NewAction(r.Id, enums.RESPONSE_CODES.UNAUTHORIZED, r.Action, nil).Dispatch(c)
      return
    }

    user, err := db.GetUser(bson.M{"id": data.Id})
    if err != nil {
      go NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    globalBan := db.NewGlobalBan(user.Id, c.U.Id, data.Reason, data.Duration)
    if err := globalBan.Save(); err != nil {
      go NewAction(r.Id, enums.RESPONSE_CODES.ERROR, r.Action, nil).Dispatch(c)
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
    c.Lock()
    defer c.Unlock()

    var data struct {
      Start bool `json:"start"`
    }

    if err := json.Unmarshal(r.Data, &data); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    if c.U.GlobalRole < enums.GLOBAL_ROLES.ADMIN {
      go NewAction(r.Id, enums.RESPONSE_CODES.UNAUTHORIZED, r.Action, nil).Dispatch(c)
      return
    }

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
  // Needs to be updated
  case "chat.send":
    var data struct {
      Me      bool   `json:"me"`
      Message string `json:"message"`
    }

    if err := json.Unmarshal(r.Data, &data); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    communityData, err := db.GetCommunity(bson.M{"id": c.Community})
    if err == mgo.ErrNotFound {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    } else if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.ERROR, r.Action, nil).Dispatch(c)
      return
    }

    community := NewCommunity(communityData)

    if len(data.Message) > 255 {
      data.Message = data.Message[:255]
    }

    community.Emit(NewEvent("chat.receive", data.Message))
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
      NewAction(r.Id, enums.RESPONSE_CODES.UNAUTHORIZED, r.Action, nil).Dispatch(c)
      return
    }

    community, err := db.NewCommunity(c.U.Id, data.Url, data.Name, data.Nsfw)
    if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    if err := community.Save(); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.ERROR, r.Action, nil).Dispatch(c)
      return
    }

    staff := db.NewCommunityStaff(community.Id, c.U.Id, enums.MODERATION_ROLES.HOST)
    if err := staff.Save(); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.ERROR, r.Action, nil).Dispatch(c)
      return
    }

    NewAction(r.Id, enums.RESPONSE_CODES.OK, r.Action, community.Struct()).Dispatch(c)
  case "community.edit":
    var data struct {
      Id              string  `json:"id"`
      Name            *string `json:"name"`
      Description     *string `json:"description"`
      WelcomeMessage  *string `json:"welcomeMessage"`
      WaitlistEnabled *bool   `json:"waitlistEnabled"`
      DjRecycling     *bool   `json:"djRecycling"`
      Nsfw            *bool   `json:"nsfw"`
    }

    if err := json.Unmarshal(r.Data, &data); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    c.Lock()
    defer c.Unlock()

    communityData, err := db.GetCommunity(bson.M{"id": data.Id})
    if err == mgo.ErrNotFound {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    } else if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.ERROR, r.Action, nil).Dispatch(c)
      return
    }

    community := NewCommunity(communityData)

    // Check the user owns this community
    if !community.HasPermission(c.U, enums.MODERATION_ROLES.HOST) {
      NewAction(r.Id, enums.RESPONSE_CODES.UNAUTHORIZED, r.Action, nil).Dispatch(c)
      return
    }

    if data.Name != nil {
      name := *data.Name
      if length := len(name); length < 2 || length > 30 {
        NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
        return
      }
      communityData.Name = name
    }

    if data.Nsfw != nil {
      communityData.Nsfw = *data.Nsfw
    }

    if data.Description != nil {
      description := *data.Description
      if length := len(description); length < 2 || length > 100 {
        NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
        return
      }
      communityData.Description = description
    }

    if data.WelcomeMessage != nil {
      welcomeMessage := *data.WelcomeMessage
      if length := len(welcomeMessage); length < 2 || length > 300 {
        NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
        return
      }
      communityData.WelcomeMessage = welcomeMessage
    }

    if data.WaitlistEnabled != nil {
      communityData.WaitlistEnabled = *data.WaitlistEnabled
    }

    if data.DjRecycling != nil {
      communityData.DjRecycling = *data.DjRecycling
    }

    // Save community
    if err := communityData.Save(); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.ERROR, r.Action, nil).Dispatch(c)
      return
    }

    NewAction(r.Id, enums.RESPONSE_CODES.OK, r.Action, communityData.Struct()).Dispatch(c)
  case "community.getHistory":
    var data struct {
      Id string `json:"id"`
    }

    if err := json.Unmarshal(r.Data, &data); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    communityData, err := db.GetCommunity(bson.M{"id": data.Id})
    if err == mgo.ErrNotFound {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    } else if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.ERROR, r.Action, nil).Dispatch(c)
      return
    }

    history, err := communityData.GetHistory(50)
    if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.ERROR, r.Action, nil).Dispatch(c)
      return
    }

    NewAction(r.Id, enums.RESPONSE_CODES.OK, r.Action, db.StructCommunityHistory(history)).Dispatch(c)
  case "community.getInfo":
    var data struct {
      Id string `json:"id"`
    }

    if err := json.Unmarshal(r.Data, &data); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    communityData, err := db.GetCommunity(bson.M{"id": data.Id})
    if err == mgo.ErrNotFound {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    } else if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.ERROR, r.Action, nil).Dispatch(c)
      return
    }

    NewAction(r.Id, enums.RESPONSE_CODES.OK, r.Action, communityData.Struct()).Dispatch(c)
  case "community.getStaff":
    var data struct {
      Id string `json:"id"`
    }

    if err := json.Unmarshal(r.Data, &data); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    communityData, err := db.GetCommunity(bson.M{"id": data.Id})
    if err == mgo.ErrNotFound {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    } else if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.ERROR, r.Action, nil).Dispatch(c)
      return
    }

    staff, err := communityData.GetStaff()
    if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.ERROR, r.Action, nil).Dispatch(c)
      return
    }

    NewAction(r.Id, enums.RESPONSE_CODES.OK, r.Action, db.StructCommunityStaff(staff)).Dispatch(c)
  case "community.getState":
    var data struct {
      Id string `json:"id"`
    }

    if err := json.Unmarshal(r.Data, &data); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    communityData, err := db.GetCommunity(bson.M{"id": data.Id})
    if err == mgo.ErrNotFound {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    } else if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.ERROR, r.Action, nil).Dispatch(c)
      return
    }

    community := NewCommunity(communityData)
    state := community.GetState()

    NewAction(r.Id, enums.RESPONSE_CODES.OK, r.Action, state).Dispatch(c)
  case "community.getUsers":
    var data struct {
      Id string `json:"id"`
    }

    if err := json.Unmarshal(r.Data, &data); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    communityData, err := db.GetCommunity(bson.M{"id": data.Id})
    if err == mgo.ErrNotFound {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    } else if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.ERROR, r.Action, nil).Dispatch(c)
      return
    }

    community := NewCommunity(communityData)
    population := community.P
    users := []string{}
    for _, u := range population {
      users = append(users, u.Id)
    }
    NewAction(r.Id, enums.RESPONSE_CODES.OK, r.Action, users)
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

    if community, ok := Communities[c.Community]; ok {
      // The return of this doesn't matter. The only error this returns is when the user isn't in the community
      _ = community.Leave(c.U)
    }

    communityData, err := db.GetCommunity(bson.M{"url": data.Url})
    if err == mgo.ErrNotFound {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    } else if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.ERROR, r.Action, nil).Dispatch(c)
      return
    }

    community := NewCommunity(communityData)
    c.Community = communityData.Id
    community.Join(c.U)
    NewAction(r.Id, enums.RESPONSE_CODES.OK, r.Action, bson.M{"id": community.C.Id}).Dispatch(c)
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
      NewAction(r.Id, enums.RESPONSE_CODES.ERROR, r.Action, nil).Dispatch(c)
      return
    }

    if len(items) <= 0 {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    communityData, err := db.GetCommunity(bson.M{"id": c.Community})
    if err == mgo.ErrNotFound {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    } else if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.ERROR, r.Action, nil).Dispatch(c)
      return
    }

    community := NewCommunity(communityData)

    NewAction(r.Id, community.JoinWaitlist(c.U), r.Action, nil).Dispatch(c)
  case "dj.leave":
    c.Lock()
    defer c.Unlock()

    communityData, err := db.GetCommunity(bson.M{"id": c.Community})
    if err == mgo.ErrNotFound {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    } else if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.ERROR, r.Action, nil).Dispatch(c)
      return
    }

    community := NewCommunity(communityData)
    NewAction(r.Id, community.LeaveWaitlist(c.U), r.Action, nil).Dispatch(c)
  case "dj.skip":
    c.Lock()
    defer c.Unlock()

    communityData, err := db.GetCommunity(bson.M{"id": c.Community})
    if err == mgo.ErrNotFound {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    } else if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.ERROR, r.Action, nil).Dispatch(c)
      return
    }

    community := NewCommunity(communityData)
    if community.M != nil && community.M.DjId == c.U.Id {
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
      Type       int    `json:"type"`
      MediaId    string `json:"mid"`
      PlaylistId string `json:"playlistId"`
    }

    if err := json.Unmarshal(r.Data, &data); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    c.Lock()
    defer c.Unlock()

    playlist, err := db.GetPlaylist(bson.M{"id": data.PlaylistId})
    if err == mgo.ErrNotFound {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    } else if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.ERROR, r.Action, nil).Dispatch(c)
      return
    }

    if playlist.OwnerId != c.U.Id {
      NewAction(r.Id, enums.RESPONSE_CODES.UNAUTHORIZED, r.Action, nil).Dispatch(c)
      return
    }

    playlistItems, err := playlist.GetItems()
    if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.ERROR, r.Action, nil).Dispatch(c)
      return
    }

    if len(playlistItems) >= 200 {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    for _, item := range playlistItems {
      if item.MediaId == data.MediaId {
        NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
        return
      }
    }

    // Add other error reporting
    media, err := db.NewMedia(data.MediaId, data.Type)
    if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.ERROR, r.Action, nil).Dispatch(c)
      return
    }

    if err := media.Save(); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.ERROR, r.Action, nil).Dispatch(c)
      return
    }

    playlistItem := db.NewPlaylistItem(data.PlaylistId, media.Title, media.Artist, data.MediaId)

    playlistItems = append([]db.PlaylistItem{playlistItem}, playlistItems...)
    if err := playlist.SaveItems(playlistItems); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.ERROR, r.Action, nil).Dispatch(c)
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
      NewAction(r.Id, enums.RESPONSE_CODES.ERROR, r.Action, nil).Dispatch(c)
      return
    }

    playlist, err := db.NewPlaylist(data.PlaylistName, c.U.Id, true)
    if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    if len(playlists) >= 25 {
      NewAction(r.Id, enums.RESPONSE_CODES.UNAUTHORIZED, r.Action, nil).Dispatch(c)
      return
    }

    if err := playlist.Save(); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.ERROR, r.Action, nil).Dispatch(c)
      return
    }

    if err := playlist.Select(c.U); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.ERROR, r.Action, nil).Dispatch(c)
      return
    }

    c.Unlock()

    // How it works
    //  Basically, we define a few things first. The amount passed,
    //  the amount failed and a map to indicate what items have been completed.
    //
    //  We then make sure that we only import a max of 200 items and begin.
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

    if total > 200 {
      failed = total - 200
      total = 200
      data.Items = data.Items[:200]
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

        playlistItem := db.NewPlaylistItem(playlist.Id, media.Title, media.Artist, item.MediaId)
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
      NewAction(r.Id, enums.RESPONSE_CODES.ERROR, r.Action, nil).Dispatch(c)
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
      results, err = searcher.SearchYoutube(data.Query)
    case 1:
      results, err = searcher.SearchSoundcloud(data.Query)
    default:
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.ERROR, r.Action, nil).Dispatch(c)
      return
    }

    NewAction(r.Id, enums.RESPONSE_CODES.OK, r.Action, results).Dispatch(c)
  /*
     Moderation
  */
  case "moderation.addDj":
    var data struct {
      UserId string `json:"userId"`
    }

    if err := json.Unmarshal(r.Data, &data); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    c.Lock()
    defer c.Unlock()

    communityData, err := db.GetCommunity(bson.M{"id": c.Community})
    if err == mgo.ErrNotFound {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    } else if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.ERROR, r.Action, nil).Dispatch(c)
      return
    }

    community := NewCommunity(communityData)

    if !community.HasPermission(c.U, enums.MODERATION_ROLES.BOUNCER) {
      NewAction(r.Id, enums.RESPONSE_CODES.UNAUTHORIZED, r.Action, nil).Dispatch(c)
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
  case "moderation.clearChat":
    c.Lock()
    defer c.Unlock()

    communityData, err := db.GetCommunity(bson.M{"id": c.Community})
    if err == mgo.ErrNotFound {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    } else if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.ERROR, r.Action, nil).Dispatch(c)
      return
    }

    community := NewCommunity(communityData)

    if !community.HasPermission(c.U, enums.MODERATION_ROLES.MANAGER) {
      NewAction(r.Id, enums.RESPONSE_CODES.UNAUTHORIZED, r.Action, nil).Dispatch(c)
      return
    }

    community.Emit(NewEvent("chat.clear", nil))
  case "moderation.deleteChat":
    var data struct {
      Id string `json:"id"`
    }

    if err := json.Unmarshal(r.Data, &data); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    c.Lock()
    defer c.Unlock()

    communityData, err := db.GetCommunity(bson.M{"id": c.Community})
    if err == mgo.ErrNotFound {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    } else if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.ERROR, r.Action, nil).Dispatch(c)
      return
    }

    community := NewCommunity(communityData)

    if !community.HasPermission(c.U, enums.MODERATION_ROLES.BOUNCER) {
      NewAction(r.Id, enums.RESPONSE_CODES.UNAUTHORIZED, r.Action, nil).Dispatch(c)
      return
    }

    community.Emit(NewEvent("chat.delete", bson.M{
      "id":      data.Id,
      "deleter": c.U.Id,
    }))
  case "moderation.forceSkip":
    c.Lock()
    defer c.Unlock()

    communityData, err := db.GetCommunity(bson.M{"id": c.Community})
    if err == mgo.ErrNotFound {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    } else if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.ERROR, r.Action, nil).Dispatch(c)
      return
    }

    community := NewCommunity(communityData)

    if !community.HasPermission(c.U, enums.MODERATION_ROLES.BOUNCER) {
      NewAction(r.Id, enums.RESPONSE_CODES.UNAUTHORIZED, r.Action, nil).Dispatch(c)
      return
    }

    community.Advance()
  case "moderation.kick":
    var data struct {
      UserId string `json:"userId"`
    }

    if err := json.Unmarshal(r.Data, &data); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    c.Lock()
    defer c.Unlock()

    communityData, err := db.GetCommunity(bson.M{"id": c.Community})
    if err == mgo.ErrNotFound {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    } else if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.ERROR, r.Action, nil).Dispatch(c)
      return
    }

    community := NewCommunity(communityData)

    if !community.HasPermission(c.U, enums.MODERATION_ROLES.BOUNCER) {
      NewAction(r.Id, enums.RESPONSE_CODES.UNAUTHORIZED, r.Action, nil).Dispatch(c)
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
    client.Unlock()

    go community.Emit(NewEvent("moderation.kick", struct {
      UserId string `json:"userId"`
    }{data.UserId}))

    NewAction(r.Id, rCode, r.Action, nil)
  case "moderation.moveDj":
    var data struct {
      UserId   string `json:"userId"`
      Position int    `json:"position"`
    }

    if err := json.Unmarshal(r.Data, &data); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    c.Lock()
    defer c.Unlock()

    communityData, err := db.GetCommunity(bson.M{"id": c.Community})
    if err == mgo.ErrNotFound {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    } else if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.ERROR, r.Action, nil).Dispatch(c)
      return
    }

    community := NewCommunity(communityData)

    if !community.HasPermission(c.U, enums.MODERATION_ROLES.MANAGER) {
      NewAction(r.Id, enums.RESPONSE_CODES.UNAUTHORIZED, r.Action, nil).Dispatch(c)
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
  case "moderation.removeDj":
    var data struct {
      UserId string `json:"userId"`
    }

    if err := json.Unmarshal(r.Data, &data); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    c.Lock()
    defer c.Unlock()

    communityData, err := db.GetCommunity(bson.M{"id": c.Community})
    if err == mgo.ErrNotFound {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    } else if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.ERROR, r.Action, nil).Dispatch(c)
      return
    }

    community := NewCommunity(communityData)

    if !community.HasPermission(c.U, enums.MODERATION_ROLES.BOUNCER) {
      NewAction(r.Id, enums.RESPONSE_CODES.UNAUTHORIZED, r.Action, nil).Dispatch(c)
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
      UserId string `json:"userId"`
      Role   int    `json:"role"`
    }

    if err := json.Unmarshal(r.Data, &data); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    c.Lock()
    defer c.Unlock()

    communityData, err := db.GetCommunity(bson.M{"id": c.Community})
    if err == mgo.ErrNotFound {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    } else if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.ERROR, r.Action, nil).Dispatch(c)
      return
    }

    community := NewCommunity(communityData)

    community.Lock()
    defer community.Unlock()

    // Check the role isn't out of biynds
    if data.Role < enums.MODERATION_ROLES.USER || data.Role > enums.MODERATION_ROLES.HOST {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    if !community.HasPermission(c.U, data.Role+1) {
      NewAction(r.Id, enums.RESPONSE_CODES.UNAUTHORIZED, r.Action, nil).Dispatch(c)
      return
    }

    user := community.GetUser(data.UserId)
    if user == nil {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    cs, err := db.GetCommunityStaff(bson.M{"communityId": c.Community, "userId": data.UserId})
    if err == mgo.ErrNotFound {
      cs = db.NewCommunityStaff(communityData.Id, user.Id, data.Role)
    } else if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.ERROR, r.Action, nil).Dispatch(c)
      return
    }

    if !community.HasPermission(c.U, cs.Role+1) {
      NewAction(r.Id, enums.RESPONSE_CODES.UNAUTHORIZED, r.Action, nil).Dispatch(c)
      return
    }

    cs.Role = data.Role

    if cs.Role <= enums.MODERATION_ROLES.USER {
      err = cs.Delete()
    } else {
      err = cs.Save()
    }

    if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.ERROR, r.Action, nil).Dispatch(c)
      return
    }

    NewAction(r.Id, enums.RESPONSE_CODES.OK, r.Action, nil).Dispatch(c)
  /*
     Playlist
  */
  case "playlist.activate":
    var data struct {
      PlaylistId string `json:"playlistId"`
    }

    if err := json.Unmarshal(r.Data, &data); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    c.Lock()
    defer c.Unlock()

    playlist, err := db.GetPlaylist(bson.M{"id": data.PlaylistId})
    if err == mgo.ErrNotFound {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    } else if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.ERROR, r.Action, nil).Dispatch(c)
      return
    }

    if playlist.OwnerId != c.U.Id {
      NewAction(r.Id, enums.RESPONSE_CODES.UNAUTHORIZED, r.Action, nil).Dispatch(c)
      return
    }

    if err := playlist.Select(c.U); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.ERROR, r.Action, nil).Dispatch(c)
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
      NewAction(r.Id, enums.RESPONSE_CODES.ERROR, r.Action, nil).Dispatch(c)
      return
    }

    playlist, err := db.NewPlaylist(data.Name, c.U.Id, true)
    if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    if len(playlists) >= 25 {
      NewAction(r.Id, enums.RESPONSE_CODES.UNAUTHORIZED, r.Action, nil).Dispatch(c)
      return
    }

    if err := playlist.Save(); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.ERROR, r.Action, nil).Dispatch(c)
      return
    }

    if err := playlist.Select(c.U); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.ERROR, r.Action, nil).Dispatch(c)
      return
    }

    NewAction(r.Id, enums.RESPONSE_CODES.OK, r.Action, playlist.Struct()).Dispatch(c)
  case "playlist.delete":
    var data struct {
      PlaylistId string `json:"playlistId"`
    }

    if err := json.Unmarshal(r.Data, &data); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    c.Lock()
    defer c.Unlock()

    playlist, err := db.GetPlaylist(bson.M{"id": data.PlaylistId})
    if err == mgo.ErrNotFound {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    } else if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.ERROR, r.Action, nil).Dispatch(c)
      return
    }

    if playlist.OwnerId != c.U.Id {
      NewAction(r.Id, enums.RESPONSE_CODES.UNAUTHORIZED, r.Action, nil).Dispatch(c)
      return
    }

    if playlist.Selected {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    if err := playlist.Delete(); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.ERROR, r.Action, nil).Dispatch(c)
      return
    }

    NewAction(r.Id, enums.RESPONSE_CODES.OK, r.Action, nil).Dispatch(c)
  case "playlist.edit":
    var data struct {
      PlaylistId string  `json:"playlistId"`
      Name       *string `json:"name"`
    }

    if err := json.Unmarshal(r.Data, &data); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    c.Lock()
    defer c.Unlock()

    playlist, err := db.GetPlaylist(bson.M{"id": data.PlaylistId})
    if err == mgo.ErrNotFound {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    } else if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.ERROR, r.Action, nil).Dispatch(c)
      return
    }

    if playlist.OwnerId != c.U.Id {
      NewAction(r.Id, enums.RESPONSE_CODES.UNAUTHORIZED, r.Action, nil).Dispatch(c)
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
      NewAction(r.Id, enums.RESPONSE_CODES.ERROR, r.Action, nil).Dispatch(c)
      return
    }

    NewAction(r.Id, enums.RESPONSE_CODES.OK, r.Action, playlist.Struct()).Dispatch(c)
  case "playlist.get":
    var data struct {
      PlaylistId string `json:"playlistId"`
    }

    if err := json.Unmarshal(r.Data, &data); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    c.Lock()
    defer c.Unlock()

    playlist, err := db.GetPlaylist(bson.M{"id": data.PlaylistId})
    if err == mgo.ErrNotFound {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    } else if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.ERROR, r.Action, nil).Dispatch(c)
      return
    }

    if playlist.OwnerId != c.U.Id {
      NewAction(r.Id, enums.RESPONSE_CODES.UNAUTHORIZED, r.Action, nil).Dispatch(c)
      return
    }

    playlistItems, err := playlist.GetItems()
    if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.ERROR, r.Action, nil).Dispatch(c)
      return
    }

    NewAction(r.Id, enums.RESPONSE_CODES.OK, r.Action, db.StructPlaylistItems(playlistItems)).Dispatch(c)
  case "playlist.getList":
    c.Lock()
    defer c.Unlock()

    playlists, err := c.U.GetPlaylists()
    if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.ERROR, r.Action, nil).Dispatch(c)
      return
    }

    NewAction(r.Id, enums.RESPONSE_CODES.OK, r.Action, db.StructPlaylists(playlists)).Dispatch(c)
  /*
     PlaylistItem
  */
  case "playlistItem.delete":
    var data struct {
      PlaylistId     string `json:"playlistId"`
      PlaylistItemId string `json:"playlistItemId"`
    }

    if err := json.Unmarshal(r.Data, &data); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    c.Lock()
    defer c.Unlock()

    playlist, err := db.GetPlaylist(bson.M{"id": data.PlaylistId})
    if err == mgo.ErrNotFound {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    } else if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.ERROR, r.Action, nil).Dispatch(c)
      return
    }

    if playlist.OwnerId != c.U.Id {
      NewAction(r.Id, enums.RESPONSE_CODES.UNAUTHORIZED, r.Action, nil).Dispatch(c)
      return
    }

    playlistItems, err := playlist.GetItems()
    if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.ERROR, r.Action, nil).Dispatch(c)
      return
    }

    communityData, err := db.GetCommunity(bson.M{"id": c.Community})
    if err == mgo.ErrNotFound {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    } else if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.ERROR, r.Action, nil).Dispatch(c)
      return
    }

    community := NewCommunity(communityData)
    if community.H != nil && community.H.PlaylistItemId == data.PlaylistItemId {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    var index int
    for i, pi := range playlistItems {
      if pi.Id == data.PlaylistItemId {
        index = i
      }
    }

    //playlistItems, playlistItems[len(playlistItems)-1] = append(playlistItems[:index], playlistItems[index+1:]...), nil
    playlistItems = append(playlistItems[:index], playlistItems[:index+1]...)

    if err := playlist.SaveItems(playlistItems); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.ERROR, r.Action, nil).Dispatch(c)
      return
    }

    NewAction(r.Id, enums.RESPONSE_CODES.OK, r.Action, nil).Dispatch(c)
  case "playlistItem.edit":
    var data struct {
      PlaylistId     string  `json:"playlistId"`
      PlaylistItemId string  `json:"playlistItemId"`
      Artist         *string `json:"artist"`
      Title          *string `json:"title"`
    }

    if err := json.Unmarshal(r.Data, &data); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    c.Lock()
    defer c.Unlock()

    playlist, err := db.GetPlaylist(bson.M{"id": data.PlaylistId})
    if err == mgo.ErrNotFound {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    } else if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.ERROR, r.Action, nil).Dispatch(c)
      return
    }

    if playlist.OwnerId != c.U.Id {
      NewAction(r.Id, enums.RESPONSE_CODES.UNAUTHORIZED, r.Action, nil).Dispatch(c)
      return
    }

    playlistItem, err := db.GetPlaylistItem(bson.M{"playlistId": data.PlaylistId, "id": data.PlaylistItemId})
    if err == mgo.ErrNotFound {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    } else if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.ERROR, r.Action, nil).Dispatch(c)
      return
    }

    communityData, err := db.GetCommunity(bson.M{"id": c.Community})
    if err == mgo.ErrNotFound {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    } else if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.ERROR, r.Action, nil).Dispatch(c)
      return
    }

    community := NewCommunity(communityData)
    if community.H != nil && community.H.PlaylistItemId == data.PlaylistItemId {
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
      NewAction(r.Id, enums.RESPONSE_CODES.ERROR, r.Action, nil).Dispatch(c)
      return
    }
    NewAction(r.Id, enums.RESPONSE_CODES.OK, r.Action, nil).Dispatch(c)
  case "playlistItem.move":
    var data struct {
      PlaylistId     string `json:"playlistId"`
      PlaylistItemId string `json:"playlistItemId"`
      Position       int    `json:"position"`
    }

    if err := json.Unmarshal(r.Data, &data); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    c.Lock()
    defer c.Unlock()

    playlist, err := db.GetPlaylist(bson.M{"id": data.PlaylistId})
    if err == mgo.ErrNotFound {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    } else if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.ERROR, r.Action, nil).Dispatch(c)
      return
    }

    if playlist.OwnerId != c.U.Id {
      NewAction(r.Id, enums.RESPONSE_CODES.UNAUTHORIZED, r.Action, nil).Dispatch(c)
      return
    }

    playlistItems, err := playlist.GetItems()
    if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.ERROR, r.Action, nil).Dispatch(c)
      return
    }

    if data.Position <= 0 || data.Position > (len(playlistItems)-1) {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    var item db.PlaylistItem
    var index int = -1
    for i, pi := range playlistItems {
      if pi.Id == data.PlaylistItemId {
        index = i
        item = pi
      }
    }

    if index < 0 {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    playlistItems = append(playlistItems[:index], append([]db.PlaylistItem{item}, playlistItems[index:]...)...)

    if err := playlist.SaveItems(playlistItems); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.ERROR, r.Action, nil).Dispatch(c)
      return
    }

    NewAction(r.Id, enums.RESPONSE_CODES.OK, r.Action, nil).Dispatch(c)

  /*
     Vote
  */
  case "vote.woot":
    c.Lock()
    defer c.Unlock()

    communityData, err := db.GetCommunity(bson.M{"id": c.Community})
    if err == mgo.ErrNotFound {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    } else if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.ERROR, r.Action, nil).Dispatch(c)
      return
    }

    community := NewCommunity(communityData)
    if community.M != nil && community.M.DjId == c.U.Id {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    NewAction(r.Id, community.Vote(c.U, "woot"), r.Action, nil).Dispatch(c)
  case "vote.meh":
    c.Lock()
    defer c.Unlock()

    communityData, err := db.GetCommunity(bson.M{"id": c.Community})
    if err == mgo.ErrNotFound {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    } else if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.ERROR, r.Action, nil).Dispatch(c)
      return
    }

    community := NewCommunity(communityData)
    if community.M != nil && community.M.DjId == c.U.Id {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    NewAction(r.Id, community.Vote(c.U, "meh"), r.Action, nil).Dispatch(c)
  case "vote.save":
    var data struct {
      PlaylistId string `json:"playlistId"`
    }

    if err := json.Unmarshal(r.Data, &data); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    c.Lock()
    defer c.Unlock()

    communityData, err := db.GetCommunity(bson.M{"id": c.Community})
    if err == mgo.ErrNotFound {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    } else if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.ERROR, r.Action, nil).Dispatch(c)
      return
    }

    community := NewCommunity(communityData)
    if community.M != nil && community.M.DjId == c.U.Id {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    // Add the currently playing media to the playlist

    playlist, err := db.GetPlaylist(bson.M{"id": data.PlaylistId})
    if err == mgo.ErrNotFound {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    } else if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.ERROR, r.Action, nil).Dispatch(c)
      return
    }

    if playlist.OwnerId != c.U.Id {
      NewAction(r.Id, enums.RESPONSE_CODES.UNAUTHORIZED, r.Action, nil).Dispatch(c)
      return
    }

    playlistItems, err := playlist.GetItems()
    if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.ERROR, r.Action, nil).Dispatch(c)
      return
    }

    if len(playlistItems) >= 200 {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    // Should add mutex to prevent last second fuck ups

    for _, item := range playlistItems {
      if item.MediaId == community.M.Media.MediaId {
        NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
        return
      }
    }

    playlistItem := db.NewPlaylistItem(data.PlaylistId, community.M.Media.Title, community.M.Media.Artist, community.M.Media.MediaId)

    playlistItems = append([]db.PlaylistItem{playlistItem}, playlistItems...)
    if err := playlist.SaveItems(playlistItems); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.ERROR, r.Action, nil).Dispatch(c)
      return
    }

    NewAction(r.Id, community.Vote(c.U, "save"), r.Action, nil).Dispatch(c)
  }
}
