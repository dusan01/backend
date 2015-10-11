package structs

import ()

type CommunityInfo struct {
  Id              string `json:"id"`
  Url             string `json:"url"`
  Name            string `json:"name"`
  HostId          string `json:"hostId"`
  Description     string `json:"description"`
  WelcomeMessage  string `json:"welcomeMessage"`
  WaitlistEnabled bool   `json:"waitlistEnabled"`
  DjRecycling     bool   `json:"djRecycling"`
  Nsfw            bool   `json:"nsfw"`
}
