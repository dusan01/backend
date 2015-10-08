package db

import (
  "sync"
  "time"
)

type GlobalBan struct {
  sync.Mutex

  // Global Ban Id
  Id string `json:"id"`

  // Bannee Id
  // See /db/user/id
  BaneeId string `json:"baneeId"`

  // Banner Id
  // See /db/user/id
  BannerId string `json:"bannerId"`

  // Reason for the ban
  // Validation
  //  0-500 Characters
  Reason string `json:"reason"`

  // Until time
  Until *time.Time `json:"until"`

  // The date this objects was created in RFC 3339
  Created string `json:"created"`

  // The date this object was updated last in RFC 3339
  Updated string `json:"updated"`
}
