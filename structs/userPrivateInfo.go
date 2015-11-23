package structs

type UserPrivateInfo struct {
	UserInfo
	Diamonds   int    `json:"diamonds"`
	Email      string `json:"email"`
	FacebookId string `json:"facebookId"`
	TwitterId  string `json:"twitterId"`
}
