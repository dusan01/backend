package structs

import ()

type UserPrivateInfo struct {
  Diamonds   int    `json:"diamonds"`
  Email      string `json:"email"`
  FacebookId string `json:"facebookId"`
  TwitterId  string `json:"twitterId"`
}
