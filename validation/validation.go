package validation

import (
  "regexp"
  "strings"
)

var ReservedUsernames = []string{
  "ivan", "str_t", "ivanmoe", "ivan.moe", "ivan-moe", "ivan_moe", "str-t", "strt", "str.t",
  "thomas", "6f", "6f7262", "6fio", "6f.io", "6f-io", "6f_io",
  "adis", "funkybasic", "funky.basic", "funky_basic", "funky-basic", "funky", "basic", "adisbasic",
  "sq10", "sq10net", "sq10.net", "sq10_net", "sq10-net",
  "turn",
  "turnfm", "turn.fm", "turnf-fm", "turn_fm",
  "turndj", "turn.dj", "turn-dj", "turn_dj",
  "adm", "admin", "administrator", "administration",
  "about", "terms", "faq", "api", "support", "help", "blog", "chat",
  "u", "r", "_", "me",
  "anonymous", "guest",
  "bot",
  "server",
  "announcement",
}

func Email(email string) (valid bool) {
  for _, v := range ReservedEmails {
    if strings.HasSuffix(email, v) {
      return
    }
  }

  valid = !(len(email) > 100 ||
    !regexp.MustCompile(`@`).MatchString(email))
  return
}

var ReservedEmails = []string{
  "turn.dj",
  "turn.fm",
  "ivan.moe",
  "sq10.net",
  "6f.io",
}

func Username(username string) (valid bool) {
  for _, v := range ReservedUsernames {
    if username == v {
      return
    }
  }

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
