package atlas

import (
  "code.google.com/p/go-uuid/uuid"
  "errors"
  "golang.org/x/crypto/bcrypt"
  "gopkg.in/mgo.v2/bson"
  "hybris/db"
  "hybris/debug"
  "regexp"
  "strings"
)

type Session struct {
  Provider string
  Token    string
}

var sessions = map[string]Session{}

func NewToken(provider, accessToken string) (token string) {
  for k, v := range sessions {
    if v.Token == accessToken {
      token = k
      return
    }
  }
  token = strings.Replace(uuid.NewUUID().String(), "-", "", -1)
  sessions[token] = Session{
    provider,
    accessToken,
  }
  return
}

func AddIntegration(user *db.User, token string) error {
  session, ok := sessions[token]
  if !ok {
    go debug.Log("[atlas > AddIntegration] Invalid token: [%s]", token)
    return errors.New("Invalid token.")
  }

  switch session.Provider {
  case "facebook":
    user.FacebookToken = session.Token
  case "twitter":
    user.TwitterToken = session.Token
  }

  delete(sessions, token)

  return nil
}

func NewSocialUser(username, token string) (*db.User, error) {
  user, err := db.NewUser(username)
  if err != nil {
    go debug.Log("[atlas > NewSocialUser] Failed to create user: [%s]", err.Error())
    return nil, err
  }
  if err := AddIntegration(user, token); err != nil {
    return nil, err
  }

  return user, nil
}

func NewEmailUser(username, email, password string) (*db.User, error) {
  user, err := db.NewUser(username)
  if err != nil {
    return nil, err
  }

  email = strings.ToLower(email)
  if length := len(email); length > 100 || !regexp.MustCompile(`@`).MatchString(email) {
    go debug.Log("[atlas > NewEmailUser] Email is invalid: [%s]", email)
    return nil, errors.New("Invalid email.")
  }
  if length := len(password); length < 2 || length > 72 {
    go debug.Log("[atlas > NewEmailUser] Password is invalid")
    return nil, errors.New("Invalid password.")
  }
  if err := db.DB.C("users").Find(bson.M{"email": email}).One(nil); err == nil {
    go debug.Log("[atlas > NewEmailUser] Email already in use: [%s]", email)
    return nil, errors.New("Email taken.")
  }

  hash, err := bcrypt.GenerateFromPassword([]byte(password), 10)
  if err != nil {
    go debug.Log("[atlas > NewEmailUser] Failed to generate password hash")
    return nil, errors.New("Server error.")
  }

  user.Email = email
  user.Password = hash
  return user, nil
}
