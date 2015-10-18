package structs

import ()

type UserInfo struct {
  Id          string `json:"id"`
  Username    string `json:"username"`
  DisplayName string `json:"displayName"`
  GlobalRole  int    `json:"globalRole"`
  Points      int    `json:"points"`
  Created     string `json:"createdAt"`
  Updated     string `json:"updatedAt"`
}
