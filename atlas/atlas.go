package atlas

import (
  "code.google.com/p/go-uuid/uuid"
  "errors"
  "golang.org/x/crypto/bcrypt"
  "gopkg.in/mgo.v2/bson"
  "hybris/db"
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

func NewSocialUser(username, token string) (*db.User, error) {
  session, ok := sessions[token]
  if !ok {
    return nil, errors.New("Invalid token")
  }

  user, err := db.NewUser(username)
  if err != nil {
    return nil, err
  }
  switch session.Provider {
  case "facebook":
    user.FacebookToken = session.Token
  case "twitter":
    user.TwitterToken = session.Token
  }
  delete(sessions, token)
  return user, nil
}

func NewEmailUser(username, email, password string) (*db.User, error) {
  user, err := db.NewUser(username)
  if err != nil {
    return nil, err
  }

  email = strings.ToLower(email)
  if length := len(email); length > 100 || !regexp.MustCompile(`@`).MatchString(email) {
    return nil, errors.New("Invalid email")
  }
  if length := len(password); length < 2 || length > 72 {
    return nil, errors.New("Invalid password")
  }
  if err := db.DB.C("users").Find(bson.M{"email": email}).One(nil); err == nil {
    return nil, errors.New("Email taken")
  }

  hash, err := bcrypt.GenerateFromPassword([]byte(password), 10)
  if err != nil {
    return nil, err
  }

  user.Email = email
  user.Password = hash
  return user, nil
}

// ATLAS
// -----
//
// NewToken
//  Returns a token to be sent to the client
//
// ValidateToken
//  Validates the given token and returns required user info
