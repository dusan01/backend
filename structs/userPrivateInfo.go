package structs

type UserPrivateInfo struct {
  Diamonds      int    `json:"diamonds"`
  Email         string `json:"email"`
  FacebookToken string `json:"facebookToken"`
  TwitterToken  string `json:"twitterToken"`
}
