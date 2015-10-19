package enums

var RESPONSE_CODES = struct {
  OK,
  BAD_REQUEST,
  UNAUTHORIZED,
  ERROR int
}{
  OK:           0,
  BAD_REQUEST:  1,
  UNAUTHORIZED: 2,
  ERROR:        3,
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
  BRONXE_DONATOR,
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
  BRONXE_DONATOR:     50,
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
}{0, 1, 2, 3, 4, 5, 6}
