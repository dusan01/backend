package db

import (
  "code.google.com/p/go-uuid/uuid"
  "errors"
  "gopkg.in/mgo.v2"
  "gopkg.in/mgo.v2/bson"
  "hybris/structs"
  "regexp"
  "strings"
  "sync"
  "time"
)

type Community struct {
  sync.Mutex

  // Community Id
  Id string `json:"id"`

  // Room URL
  // Validation
  //  2-25 Characters
  //  Lowercased
  //  Must be a-z 0-9 and '-'
  Url string `json:"url"`

  // Room name
  // Validation
  //  2-30 Characters
  Name string `json:"name"`

  // The id of the host
  // db/user/id
  HostId string `json:"host_id"`

  // Community description
  // Validation
  //  0-1000 Characters
  Description string `json:"description"`

  // Welcome Message
  // Validation
  //  0-300 Characters
  WelcomeMessage string `json:"welcomeMessage"`

  // Waitlist Locked
  // Whether or not the waitlist is emabled
  WaitlistEnabled bool `json:"waitlistEnabled"`

  // Dj Rotation
  // Whether Dj Recycling is enabled or not
  DjRecycling bool `json:"djRecycling"`

  // NSFW
  // Whether or not the community plays NSFW content or not
  Nsfw bool `json:"nsfw"`

  // The date this objects was created in RFC 3339
  Created string `json:"created"`

  // The date this object was updated last in RFC 3339
  Updated string `json:"updated"`
}

func NewCommunity(host, url, name string, nsfw bool) (*Community, error) {
  url = strings.ToLower(url)

  // Validation
  if length := len(url); length < 2 || length > 25 || !regexp.MustCompile(`^[a-zA-Z0-9\-]+$`).MatchString(url) {
    return nil, errors.New("Invalid URL")
  }
  if length := len(name); length < 2 || length > 30 {
    return nil, errors.New("Invalid Name")
  }

  // Does it already exist?
  if _, err := GetCommunity(bson.M{"url": url}); err == nil {
    return nil, errors.New("Exists")
  }

  return &Community{
    Id:              strings.Replace(uuid.NewUUID().String(), "-", "", -1),
    Url:             url,
    Name:            name,
    HostId:          host,
    WaitlistEnabled: true,
    DjRecycling:     true,
    Nsfw:            nsfw,
    Created:         time.Now().Format(time.RFC3339),
    Updated:         time.Now().Format(time.RFC3339),
  }, nil
}

func GetCommunity(query interface{}) (*Community, error) {
  var c Community
  err := DB.C("communities").Find(query).One(&c)
  return &c, err
}

func (c Community) Struct() structs.CommunityInfo {
  return structs.CommunityInfo{
    Id:              c.Id,
    Url:             c.Url,
    Name:            c.Name,
    HostId:          c.HostId,
    Description:     c.Description,
    WelcomeMessage:  c.WelcomeMessage,
    WaitlistEnabled: c.WaitlistEnabled,
    DjRecycling:     c.DjRecycling,
    Nsfw:            c.Nsfw,
  }
}

func (c Community) GetStaff() ([]CommunityStaff, error) {
  var cs []CommunityStaff
  err := DB.C("staff").Find(bson.M{"communityid": c.Id}).Iter().All(&cs)
  return cs, err
}

func (c Community) GetHistory(max int) ([]CommunityHistory, error) {
  var ch []CommunityHistory
  err := DB.C("history").Find(bson.M{"communityid": c.Id}).Limit(max).Iter().All(&ch)
  return ch, err
}

func (c Community) Save() error {
  err := DB.C("communities").Update(bson.M{"id": c.Id}, c)
  if err == mgo.ErrNotFound {
    return DB.C("communities").Insert(c)
  }
  return err
}
