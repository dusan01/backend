package structs

import ()

type UserInfo struct {
  Id          string `json:"id"`
  Username    string `json:"username"`
  DisplayName string `json:"displayName"`
  Password    []byte `json:"password"`
  GlobalRole  int    `json:"globalRole"`
  Points      int    `json:"points"`
  Created     string `json:"created"`
  Updated     string `json:"updated"`
}
