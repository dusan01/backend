package pool

import (
  "encoding/json"
  "github.com/gorilla/websocket"
  "gopkg.in/mgo.v2"
  "gopkg.in/mgo.v2/bson"
  "hybris/db"
  "hybris/enums"
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
    community.Leave(c.U)
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
    c.Terminate()
  }
}

func (c *Client) Receive(msg []byte) {
  var r struct {
    Id     int             `json:"i"`
    Action string          `json:"a"`
    Data   json.RawMessage `json:"d"`
  }

  if err := json.Unmarshal(msg, &r); err != nil {
    return
  }

  switch r.Action {
  case "echo":
    NewAction(r.Id, enums.RESPONSE_CODES.OK, r.Action, nil).Dispatch(c)

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

    _ = json.Unmarshal(r.Data, &data)

    if c.U.GlobalRole < enums.GLOBAL_ROLES.ADMIN {
      go NewAction(r.Id, enums.RESPONSE_CODES.UNAUTHORIZED, r.Action, nil).Dispatch(c)
      return
    }

    event := NewEvent(r.Action, data)
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

    _ = json.Unmarshal(r.Data, &data)

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
      go client.Terminate()
    }

    NewAction(r.Id, enums.RESPONSE_CODES.OK, r.Action, nil).Dispatch(c)
  case "adm.maintenance":
    c.Lock()
    defer c.Unlock()

    var data struct {
      Start bool `json:"start"`
    }

    _ = json.Unmarshal(r.Data, &data)

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

    _ = json.Unmarshal(r.Data, &data)

    if c.U.GlobalRole < 2 {
      NewAction(r.Id, enums.RESPONSE_CODES.UNAUTHORIZED, r.Action, nil).Dispatch(c)
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

    _ = json.Unmarshal(r.Data, &data)

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

    NewAction(r.Id, enums.RESPONSE_CODES.OK, r.Action, community.Struct()).Dispatch(c)
  case "community.edit":
    var data struct {
      Url             string  `json:"url"`
      Name            *string `json:"name"`
      Nsfw            *bool   `json:"nsfw"`
      Description     *string `json:"description"`
      WelcomeMessage  *string `json:"welcomeMessage"`
      WaitlistEnabled *bool   `json:"waitlistEnabled"`
      DjRecycling     *bool   `json:"djRecycling"`
    }

    _ = json.Unmarshal(r.Data, &data)

    c.Lock()
    defer c.Unlock()

    community, err := db.GetCommunity(bson.M{"url": data.Url})
    if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    }

    // Check the user owns this community
    if !community.HasPermission(c.U.GlobalRole, c.U.Id, 5) {
      NewAction(r.Id, enums.RESPONSE_CODES.UNAUTHORIZED, r.Action, nil).Dispatch(c)
      return
    }

    if data.Name != nil {
      name := *data.Name
      if length := len(name); length < 2 || length > 30 {
        NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
        return
      }
      community.Name = name
    }

    if data.Nsfw != nil {
      community.Nsfw = *data.Nsfw
    }

    if data.Description != nil {
      description := *data.Description
      if length := len(description); length < 2 || length > 100 {
        NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
        return
      }
      community.Description = description
    }

    if data.WelcomeMessage != nil {
      welcomeMessage := *data.WelcomeMessage
      if length := len(welcomeMessage); length < 2 || length > 300 {
        NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
        return
      }
      community.WelcomeMessage = welcomeMessage
    }

    if data.WaitlistEnabled != nil {
      community.WaitlistEnabled = *data.WaitlistEnabled
    }

    if data.DjRecycling != nil {
      community.DjRecycling = *data.DjRecycling
    }

    // Save community
    if err := community.Save(); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.ERROR, r.Action, nil).Dispatch(c)
      return
    }

    NewAction(r.Id, enums.RESPONSE_CODES.OK, r.Action, community.Struct()).Dispatch(c)
  case "community.getHistory":
    var data struct {
      Url string `json:"url"`
    }

    _ = json.Unmarshal(r.Data, &data)

    communityData, err := db.GetCommunity(bson.M{"url": data.Url})
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
      Url string `json:"url"`
    }

    _ = json.Unmarshal(r.Data, &data)

    communityData, err := db.GetCommunity(bson.M{"url": data.Url})
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
      Url string `json:"url"`
    }

    _ = json.Unmarshal(r.Data, &data)

    communityData, err := db.GetCommunity(bson.M{"url": data.Url})
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
      Url string `json:"url"`
    }

    _ = json.Unmarshal(r.Data, &data)

    communityData, err := db.GetCommunity(bson.M{"url": data.Url})
    if err == mgo.ErrNotFound {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
      return
    } else if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.ERROR, r.Action, nil).Dispatch(c)
      return
    }

    community := NewCommunity(communityData)
    state := structs.CommunityState{
      Waitlist: community.W,
    }
    if community.M != nil {
      state.NowPlaying = community.M.Struct()
    } else {
      state.NowPlaying = structs.HistoryItem{}
    }

    NewAction(r.Id, enums.RESPONSE_CODES.OK, r.Action, state).Dispatch(c)
  case "community.getUsers":
    var data struct {
      Url string `json:"url"`
    }

    _ = json.Unmarshal(r.Data, &data)

    communityData, err := db.GetCommunity(bson.M{"url": data.Url})
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

    _ = json.Unmarshal(r.Data, &data)

    c.Lock()
    defer c.Unlock()

    if community, ok := Communities[c.Community]; ok {
      community.Leave(c.U)
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
    state := structs.CommunityState{
      Waitlist: community.W,
    }
    if community.M != nil {
      state.NowPlaying = community.M.Struct()
    } else {
      state.NowPlaying = structs.HistoryItem{}
    }

    c.Community = communityData.Id
    community.Join(c.U)
    NewAction(r.Id, enums.RESPONSE_CODES.OK, r.Action, nil).Dispatch(c)
  case "community.taken":
    var data struct {
      Url string `json:"url"`
    }

    _ = json.Unmarshal(r.Data, &data)

    _, err := db.GetCommunity(bson.M{"url": data.Url})
    NewAction(r.Id, enums.RESPONSE_CODES.OK, r.Action, bson.M{"taken": err == nil})
  /*
     Dj
  */
  case "dj.join":
    c.Lock()
    defer c.Unlock()

    if c.U.GlobalRole < 2 {
      NewAction(r.Id, enums.RESPONSE_CODES.UNAUTHORIZED, r.Action, nil).Dispatch(c)
      return
    }

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
    community.JoinWaitlist(c.U)

    NewAction(r.Id, enums.RESPONSE_CODES.OK, r.Action, nil).Dispatch(c)
  case "dj.leave":
    c.Lock()
    defer c.Unlock()

    if c.U.GlobalRole < 2 {
      NewAction(r.Id, enums.RESPONSE_CODES.UNAUTHORIZED, r.Action, nil).Dispatch(c)
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
    community.LeaveWaitlist(c.U)
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
    if community.M != nil && community.M.UserId == c.U.Id {
      community.Advance()
      NewAction(r.Id, enums.RESPONSE_CODES.OK, r.Action, nil)
      return
    }
    NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil)

  /*
     Media
  */
  case "media.add":
    var data struct {
      Type       int    `json:"type"`
      MediaId    string `json:"mid"`
      PlaylistId string `json:"playlistId"`
    }

    _ = json.Unmarshal(r.Data, &data)

    c.Lock()
    defer c.Unlock()

    if c.U.GlobalRole < 2 {
      NewAction(r.Id, enums.RESPONSE_CODES.UNAUTHORIZED, r.Action, nil).Dispatch(c)
      return
    }

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

    item := db.NewPlaylistItem(data.PlaylistId, media.Title, media.Artist, data.MediaId)

    playlistItems = append([]db.PlaylistItem{item}, playlistItems...)
    if err := playlist.SaveItems(playlistItems); err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.ERROR, r.Action, nil).Dispatch(c)
      return
    }

    NewAction(r.Id, enums.RESPONSE_CODES.OK, r.Action, nil).Dispatch(c)
  case "media.import":
  case "media.search":
  /*
     Moderation
  */
  case "moderation.addDj":
  case "moderation.ban":
  case "moderation.clearChat":
  case "moderation.deleteChat":
  case "moderation.forceSkip":
  case "moderation.kick":
  case "moderation.moveDj":
  case "moderation.mute":
  case "moderation.removeDj":
  case "moderation.setRole":
  /*
     Playlist
  */
  case "playlist.activate":
    var data struct {
      PlaylistId string `json:"playlistId"`
    }

    _ = json.Unmarshal(r.Data, &data)

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
      Name  string `json:"name"`
      Title string `json:"title"`
    }

    _ = json.Unmarshal(r.Data, &data)

    c.Lock()
    defer c.Unlock()

    if c.U.GlobalRole < 2 {
      NewAction(r.Id, enums.RESPONSE_CODES.UNAUTHORIZED, r.Action, nil).Dispatch(c)
      return
    }

    playlists, err := c.U.GetPlaylists()
    if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.ERROR, r.Action, nil).Dispatch(c)
      return
    }

    if len(playlists) >= 25 {
      NewAction(r.Id, enums.RESPONSE_CODES.UNAUTHORIZED, r.Action, nil).Dispatch(c)
      return
    }

    playlist, err := db.NewPlaylist(data.Name, c.U.Id, true)
    if err != nil {
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
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

    _ = json.Unmarshal(r.Data, &data)

    c.Lock()
    defer c.Unlock()

    if c.U.GlobalRole < 2 {
      NewAction(r.Id, enums.RESPONSE_CODES.UNAUTHORIZED, r.Action, nil).Dispatch(c)
      return
    }

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
  case "playlist.get":
    var data struct {
      PlaylistId string `json:"playlistId"`
    }

    _ = json.Unmarshal(r.Data, &data)

    c.Lock()
    defer c.Unlock()

    if c.U.GlobalRole < 2 {
      NewAction(r.Id, enums.RESPONSE_CODES.UNAUTHORIZED, r.Action, nil).Dispatch(c)
      return
    }

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
  case "playlist.getlist":
    c.Lock()
    defer c.Unlock()

    if c.U.GlobalRole < 2 {
      NewAction(r.Id, enums.RESPONSE_CODES.UNAUTHORIZED, r.Action, nil).Dispatch(c)
      return
    }

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

    _ = json.Unmarshal(r.Data, &data)

    c.Lock()
    defer c.Unlock()

    if c.U.GlobalRole < 2 {
      NewAction(r.Id, enums.RESPONSE_CODES.UNAUTHORIZED, r.Action, nil).Dispatch(c)
      return
    }

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
    if community.M != nil && community.M.PlaylistItemId == data.PlaylistItemId {
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

    _ = json.Unmarshal(r.Data, &data)

    c.Lock()
    defer c.Unlock()

    if c.U.GlobalRole < 2 {
      NewAction(r.Id, enums.RESPONSE_CODES.UNAUTHORIZED, r.Action, nil).Dispatch(c)
      return
    }

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
    if community.M != nil && community.M.PlaylistItemId == data.PlaylistItemId {
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

    _ = json.Unmarshal(r.Data, &data)

    c.Lock()
    defer c.Unlock()

    if c.U.GlobalRole < 2 {
      NewAction(r.Id, enums.RESPONSE_CODES.UNAUTHORIZED, r.Action, nil).Dispatch(c)
      return
    }

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
    */ /*
       case "vote.woot":
         c.Lock()
         defer c.Unlock()

         if c.U.GlobalRole < 2 {
           NewAction(r.Id, enums.RESPONSE_CODES.UNAUTHORIZED, r.Action, nil).Dispatch(c)
           return
         }

         ommunityData, err := db.GetCommunity(bson.M{"id": c.Community})
         if err == mgo.ErrNotFound {
           NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
           return
         } else if err != nil {
           NewAction(r.Id, enums.RESPONSE_CODES.ERROR, r.Action, nil).Dispatch(c)
           return
         }

         community := NewCommunity(communityData)
         if community.M != nil && community.M.PlaylistItemId == data.PlaylistItemId {
           NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
           return
         }

         community.Vote(1)
       case "vote.meh":
         c.Lock()
         defer c.Unlock()

         if c.U.GlobalRole < 2 {
           NewAction(r.Id, enums.RESPONSE_CODES.UNAUTHORIZED, r.Action, nil).Dispatch(c)
           return
         }

         ommunityData, err := db.GetCommunity(bson.M{"id": c.Community})
         if err == mgo.ErrNotFound {
           NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
           return
         } else if err != nil {
           NewAction(r.Id, enums.RESPONSE_CODES.ERROR, r.Action, nil).Dispatch(c)
           return
         }

         community := NewCommunity(communityData)
         if community.M != nil && community.M.PlaylistItemId == data.PlaylistItemId {
           NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
           return
         }

         community.Vote(-1)
       case "vote.save":
         var data struct {
           PlaylistItemId string `json:"playlistItemId"`
         }

         _ = json.Unmarshal(r.Data, &data)

         c.Lock()
         defer c.Unlock()

         if c.U.GlobalRole < 2 {
           NewAction(r.Id, enums.RESPONSE_CODES.UNAUTHORIZED, r.Action, nil).Dispatch(c)
           return
         }

         ommunityData, err := db.GetCommunity(bson.M{"id": c.Community})
         if err == mgo.ErrNotFound {
           NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
           return
         } else if err != nil {
           NewAction(r.Id, enums.RESPONSE_CODES.ERROR, r.Action, nil).Dispatch(c)
           return
         }

         community := NewCommunity(communityData)
         if community.M != nil && community.M.PlaylistItemId == data.PlaylistItemId {
           NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(c)
           return
         }



         community.Vote(0, c.U)*/
  }
}
