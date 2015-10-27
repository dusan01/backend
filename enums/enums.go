package enums

const (
  VERSION = "0.1.0"
)

var RESPONSE_CODES = struct {
  OK,
  BAD_REQUEST,
  NOT_FOUND,
  FORBIDDEN,
  ALREADY_LOGGED_IN,
  UNIMPLEMENTED,
  SERVER_ERROR int
}{
  OK:                0,
  BAD_REQUEST:       1,
  NOT_FOUND:         2,
  FORBIDDEN:         3,
  ALREADY_LOGGED_IN: 100,
  UNIMPLEMENTED:     9998,
  SERVER_ERROR:      9999,
}

var BROADCAST_TYPES = struct {
  SYSTEM_ALERT,
  SYSTEM_ANNOUNCEMENT,
  GLOBAL_CHAT,
  ADVERTISEMENT int
}{
  SYSTEM_ALERT:        0,
  SYSTEM_ANNOUNCEMENT: 1,
  GLOBAL_CHAT:         2,
  ADVERTISEMENT:       3,
}

var GLOBAL_ROLES = struct {
  DUMMY,
  GUEST,
  USER,
  BRONZE_DONATOR,
  SILVER_DONATOR,
  GOLD_DONATOR,
  PLATINUM_DONATOR,
  TRIAL_AMBASSADOR,
  AMBASSADOR,
  TRUSTED_AMBASSADOR,
  ADMIN,
  SERVER int
}{
  DUMMY:              0,
  GUEST:              1,
  USER:               2,
  BRONZE_DONATOR:     50,
  SILVER_DONATOR:     51,
  GOLD_DONATOR:       52,
  PLATINUM_DONATOR:   53,
  TRIAL_AMBASSADOR:   400,
  AMBASSADOR:         401,
  TRUSTED_AMBASSADOR: 402,
  ADMIN:              999,
  SERVER:             1000,
}

var MODERATION_ROLES = struct {
  GUEST,
  USER,
  DJ,
  BOUNCER,
  MANAGER,
  COHOST,
  HOST int
}{
  GUEST:   0,
  USER:    1,
  DJ:      2,
  BOUNCER: 3,
  MANAGER: 4,
  COHOST:  5,
  HOST:    6,
}
