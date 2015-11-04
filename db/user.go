package db

import (
  "errors"
  "gopkg.in/mgo.v2/bson"
  "hybris/debug"
  "hybris/structs"
  "hybris/validation"
  "strings"
  "time"
)

type User struct {
  // User Id
  Id bson.ObjectId `json:"id" bson:"_id"`

  // The user's username (all in lowercase)
  // Validation
  //  2-20 Characters
  //  A-Z a-z 0-9 '.' '-' and '_'
  Username string `json:"username" bson:"username"`

  // The user's display name
  // As it stands, it's identical to the username but with the casing preserved.
  DisplayName string `json:"displayName" bson:"displayName"`

  // The user's email
  // Validation
  //  100 characters max
  //  Must conform to ^\w+@[a-zA-Z_]+?\.[a-zA-Z]{2,3}$
  Email string `json:"email" bson:"email"`

  // The bcrypted hash of the password
  // Validation
  //  2-72 Characters
  Password []byte `json:"password" bson:"password"`

  // SEE enum/GLOBAL_ROLE/USER
  // The user's global role
  // DEFAULT 2
  GlobalRole int `json:"globalRole" bson:"globalRole"`

  // Ammount of DJ points they have
  Points int `json:"points" bson:"points"`

  // Facebook user ID used for facebook logins
  FacebookId string `json:"facebookId" bson:"facebookId"`

  // Twitter user ID used for twitter logins
  TwitterId string `json:"twitterId" bson:"twitterId"`

  // A premium currency
  // Used to purchase fancy items and features
  Diamonds int `json:"diamonds" bson:"diamonds"`

  // The date this objects was created
  Created time.Time `json:"created" bson:"created"`

  // The date this object was updated last
  Updated time.Time `json:"updated" bson:"updated"`
}

func NewUser(username string) (*User, error) {
  c := DB.C("users")
  displayName := username
  username = strings.ToLower(username)

  // Validate info
  if !validation.Username(username) {
    debug.Log("[db/NewUser] Invalid username: %s", username)
    return nil, errors.New("Invalid username.")
  }

  // Check exists
  if err := c.Find(bson.M{"username": username}).One(nil); err == nil {
    debug.Log("[db/NewUser] Username taken: %s", username)
    return nil, errors.New("Username taken.")
  }

  u := &User{
    Id:          bson.NewObjectId(),
    Username:    username,
    DisplayName: displayName,
    GlobalRole:  2,
    Created:     time.Now(),
    Updated:     time.Now(),
  }
  return u, nil
}

func GetUser(query interface{}) (*User, error) {
  var u User
  err := DB.C("users").Find(query).One(&u)
  return &u, err
}

func (u User) Save() error {
  u.Updated = time.Now()
  _, err := DB.C("users").UpsertId(u.Id, u)
  return err
}

func (u User) GetCommunities() ([]Community, error) {
  var communities []Community
  err := DB.C("communities").Find(bson.M{"host": u.Id}).Sort("-$natural").Iter().All(&communities)
  return communities, err
}

func (u User) GetActivePlaylist() (*Playlist, error) {
  return GetPlaylist(bson.M{"ownerId": u.Id, "selected": true})
}

func (u User) GetPlaylists() ([]Playlist, error) {
  var playlists []Playlist
  err := DB.C("playlists").Find(bson.M{"ownerId": u.Id}).Sort("-$natural").Iter().All(&playlists)
  return u.sortPlaylists(playlists), err
}

func (u User) SavePlaylists(playlists []Playlist) error {
  playlists = u.recalculatePlaylists(playlists)
  for _, playlist := range playlists {
    if err := playlist.Save(); err != nil {
      return err
    }
  }
  return nil
}

func (u User) sortPlaylists(playlists []Playlist) []Playlist {
  payload := make([]Playlist, len(playlists))
  for _, v := range playlists {
    payload[v.Order] = v
  }
  return payload
}

func (u User) recalculatePlaylists(playlists []Playlist) []Playlist {
  payload := []Playlist{}
  for i, playlist := range playlists {
    playlist.Order = i
    payload = append(payload, playlist)
  }
  return payload
}

func (u User) Struct() structs.UserInfo {
  return structs.UserInfo{
    Id:          u.Id,
    Username:    u.Username,
    DisplayName: u.DisplayName,
    GlobalRole:  u.GlobalRole,
    Points:      u.Points,
    Created:     u.Created,
    Updated:     u.Updated,
  }
}

func (u User) PrivateStruct() structs.UserPrivateInfo {
  return structs.UserPrivateInfo{
    u.Struct(),
    u.Diamonds,
    u.Email,
    u.FacebookId,
    u.TwitterId,
  }
}
