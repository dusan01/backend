package db

import (
	"gopkg.in/mgo.v2/bson"
	"hybris/structs"
	"time"
)

type Chat struct {
	// Chat Id
	Id bson.ObjectId `json:"id" bson:"_id"`

	// User Id
	// See /db/user/id
	UserId bson.ObjectId `json:"userId" bson:"userId"`

	// Community Id
	// See /db/community/id
	CommunityId bson.ObjectId `json:"communityId" bson:"communityId"`

	// Whether it's a /me message
	Me bool `json:"me" bson:"me"`

	// Chat message
	// Validation
	//  Max 300 characters
	Message string `json:"message" bson:"message"`

	// Deleted
	// Whether or not the chat message has been deleted
	Deleted bool `json:"deleted" bson:"deleted"`

	// Deleter Id
	// The person who deleted the message
	// See /db/user/id
	DeleterId bson.ObjectId `json:"deleterId" bson:"deleterId,omitempty"`

	// The date this objects was created
	Created time.Time `json:"created" bson:"created"`

	// The date this object was updated
	Updated time.Time `json:"updated" bson:"updated"`
}

func NewChat(userId, communityId bson.ObjectId, me bool, message string) Chat {
	if len(message) > 255 {
		message = message[:255]
	}

	return Chat{
		Id:          bson.NewObjectId(),
		UserId:      userId,
		CommunityId: communityId,
		Me:          me,
		Message:     message,
		Deleted:     false,
		DeleterId:   "",
		Created:     time.Now(),
		Updated:     time.Now(),
	}
}

func GetChat(query interface{}) (*Chat, error) {
	var c Chat
	err := DB.C("chat").Find(query).One(&c)
	return &c, err
}

func (c Chat) Save() error {
	c.Updated = time.Now()
	_, err := DB.C("chat").UpsertId(c.Id, c)
	return err
}

// Soft delete
func (c Chat) Delete() error {
	c.Deleted = true
	return c.Save()
}

func (c Chat) Struct() structs.Chat {
	return structs.Chat{
		Id:      c.Id,
		UserId:  c.UserId,
		Me:      c.Me,
		Message: c.Message,
		Time:    c.Created,
	}
}
