package db

import (
  "errors"
  "gopkg.in/mgo.v2/bson"
  "hybris/structs"
  "regexp"
  "strings"
  "time"
)

type Community struct {
  // Community Id
  Id bson.ObjectId `json:"id" bson:"_id"`

  // Room URL
  // Validation
  //  2-25 Characters
  //  Lowercased
  //  Must be a-z 0-9 and '-'
  Url string `json:"url" bson:"url"`

  // Room name
  // Validation
  //  2-30 Characters
  Name string `json:"name" bson:"name"`

  // The id of the host
  // db/user/id
  HostId bson.ObjectId `json:"hostId" bson:"hostId"`

  // Community description
  // Validation
  //  0-1000 Characters
  Description string `json:"description" bson:"description"`

  // Welcome Message
  // Validation
  //  0-300 Characters
  WelcomeMessage string `json:"welcomeMessage" bson:"welcomeMessage"`

  // Waitlist Locked
  // Whether or not the waitlist is emabled
  WaitlistEnabled bool `json:"waitlistEnabled" bson:"waitlistEnabled"`

  // Dj Rotation
  // Whether Dj Recycling is enabled or not
  DjRecycling bool `json:"djRecycling" bson:"djRecycling"`

  // NSFW
  // Whether or not the community plays NSFW content or not
  Nsfw bool `json:"nsfw" bson:"nsfw"`

  // The date this objects was created
  Created time.Time `json:"created" bson:"created"`

  // The date this object was updated
  Updated time.Time `json:"updated" bson:"updated"`
}

func NewCommunity(host bson.ObjectId, url, name string, nsfw bool) (*Community, error) {
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
    Id:              bson.NewObjectId(),
    Url:             url,
    Name:            name,
    HostId:          host,
    WaitlistEnabled: true,
    DjRecycling:     true,
    Nsfw:            nsfw,
    Created:         time.Now(),
    Updated:         time.Now(),
  }, nil
}

func GetCommunity(query interface{}) (*Community, error) {
  var c Community
  err := DB.C("communities").Find(query).One(&c)
  return &c, err
}

func (c Community) GetStaff() ([]CommunityStaff, error) {
  var cs []CommunityStaff
  err := DB.C("communityStaff").Find(bson.M{"communityId": c.Id}).Iter().All(&cs)
  return cs, err
}

func (c Community) GetHistory(max int) ([]CommunityHistory, error) {
  var ch []CommunityHistory
  err := DB.C("communityHistory").Find(bson.M{"communityId": c.Id}).Limit(max).Iter().All(&ch)
  return ch, err
}

func (c Community) Save() error {
  c.Updated = time.Now()
  _, err := DB.C("communities").UpsertId(c.Id, c)
  return err
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
