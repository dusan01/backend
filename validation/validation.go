package validation

import (
  "regexp"
)

func Email(email string) (valid bool) {
  valid = !(len(email) > 100 ||
    !regexp.MustCompile(`@`).MatchString(email))
  return
}

func Username(username string) (valid bool) {
  valid = !(len(username) < 2 ||
    len(username) > 20 ||
    !regexp.MustCompile(`^[a-zA-Z0-9_\-\.]+$`).MatchString(username))
  return
}

func Password(password string) (valid bool) {
  valid = !(len(password) < 2 ||
    len(password) > 72)
  return
}
