package db

import (
  "code.google.com/p/go-uuid/uuid"
  "errors"
  "gopkg.in/mgo.v2"
  "gopkg.in/mgo.v2/bson"
  "hybris/debug"
  "hybris/structs"
  "regexp"
  "strings"
  "sync"
  "time"
)

type User struct {
  sync.Mutex

  // Primary key -- Permanent ID of the user.
  // Represented by a UUID with any '-' removed
  Id string `json:"id" bson:"id"`

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
  FacebookToken string `json:"facebookToken" bson:"facebookToken"`

  // Twitter user ID used for twitter logins
  TwitterToken string `json:"twitterToken" bson:"twitterToken"`

  // A premium currency
  // Used to purchase fancy items and features
  Diamonds int `json:"diamonds" bson:"diamonds"`

  // The date this objects was created in RFC 3339
  Created string `json:"created" bson:"created"`

  // The date this object was updated last in RFC 3339
  Updated string `json:"updated" bson:"updated"`
}

func NewUser(username string) (*User, error) {
  c := DB.C("users")
  displayName := username
  username = strings.ToLower(username)

  // Validate info
  if length := len(username); length < 2 || length > 20 || !regexp.MustCompile(`^[a-zA-Z0-9_\-\.]+$`).MatchString(username) {
    go debug.Log("[db > NewUser] Invalid username: %s", username)
    return nil, errors.New("invalid username")
  }

  // Check exists
  if err := c.Find(bson.M{"username": username}).One(nil); err == nil {
    go debug.Log("[db > NewUser] Username taken: %s", username)
    return nil, errors.New("username taken")
  }

  u := &User{
    Id:          strings.Replace(uuid.NewUUID().String(), "-", "", -1),
    Username:    username,
    DisplayName: displayName,
    GlobalRole:  2,
    Created:     time.Now().Format(time.RFC3339),
    Updated:     time.Now().Format(time.RFC3339),
  }
  return u, nil
}

func GetUser(query interface{}) (*User, error) {
  var u User
  err := DB.C("users").Find(query).One(&u)
  return &u, err
}

func (u User) Save() error {
  err := DB.C("users").Update(bson.M{"id": u.Id}, u)
  if err == mgo.ErrNotFound {
    return DB.C("users").Insert(u)
  }
  return err
}

func (u User) GetCommunities() ([]Community, error) {
  var communities []Community
  err := DB.C("communities").Find(bson.M{"host": u.Id}).Iter().All(&communities)
  return communities, err
}

func (u User) GetActivePlaylist() (*Playlist, error) {
  return GetPlaylist(bson.M{"ownerId": u.Id, "selected": true})
}

func (u User) GetPlaylists() ([]Playlist, error) {
  var playlists []Playlist
  err := DB.C("playlists").Find(bson.M{"ownerId": u.Id}).Iter().All(&playlists)
  return playlists, err
}

func (u User) Struct() interface{} {
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
