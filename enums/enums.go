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

var GLOBAL_ROLES = struct {
  DUMMY,
  GUEST,
  USER,
  TRIAL_AMBASSADOR,
  AMBASSADOR,
  ADMIN,
  SERVER int
}{
  DUMMY:            0,
  GUEST:            1,
  USER:             2,
  TRIAL_AMBASSADOR: 50,
  AMBASSADOR:       51,
  ADMIN:            999,
  SERVER:           1000,
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
