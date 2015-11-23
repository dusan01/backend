package structs

import "gopkg.in/mgo.v2/bson"

type CommunityInfo struct {
	Id              bson.ObjectId `json:"id"`
	Url             string        `json:"url"`
	Name            string        `json:"name"`
	HostId          bson.ObjectId `json:"hostId"`
	Description     string        `json:"description"`
	WelcomeMessage  string        `json:"welcomeMessage"`
	WaitlistEnabled bool          `json:"waitlistEnabled"`
	DjRecycling     bool          `json:"djRecycling"`
	Nsfw            bool          `json:"nsfw"`
}
