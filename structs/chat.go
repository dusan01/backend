package structs

type Chat struct {
  Id      string `json:"id"`
  UserId  string `json:"userId"`
  Me      bool   `json:"me"`
  Message string `json:"message"`
  Time    string `json:"time"`
}
