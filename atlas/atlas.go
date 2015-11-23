package atlas

import (
	"errors"
	"hybris/db/dbuser"
	"hybris/validation"
	"strings"

	"code.google.com/p/go-uuid/uuid"
	"golang.org/x/crypto/bcrypt"
	uppdb "upper.io/db"
)

type Session struct {
	Provider string
	UserId   string
}

var sessions = map[string]Session{}

func NewToken(provider, userId string) (token string) {
	for k, v := range sessions {
		if v.UserId == userId {
			token = k
			return
		}
	}
	token = strings.Replace(uuid.NewUUID().String(), "-", "", -1)
	sessions[token] = Session{
		provider,
		userId,
	}
	return
}

func AddIntegration(user *dbuser.User, token string) error {
	session, ok := sessions[token]
	if !ok {
		return errors.New("Invalid token.")
	}

	switch session.Provider {
	case "facebook":
		user.FacebookId = session.UserId
	case "twitter":
		user.TwitterId = session.UserId
	}

	delete(sessions, token)

	return nil
}

func NewSocialUser(username, token string) (dbuser.User, error) {
	user, err := dbuser.New(username)
	if err != nil {
		return dbuser.User{}, err
	}
	if err := AddIntegration(&user, token); err != nil {
		return dbuser.User{}, err
	}

	return user, nil
}

func NewEmailUser(username, email, password string) (dbuser.User, error) {
	user, err := dbuser.New(username)
	if err != nil {
		return dbuser.User{}, err
	}

	email = strings.ToLower(email)
	if !validation.Email(email) {
		return dbuser.User{}, errors.New("Invalid email.")
	}
	if !validation.Password(password) {
		return dbuser.User{}, errors.New("Invalid password.")
	}
	if _, err := dbuser.Get(uppdb.Cond{"email": email}); err == nil {
		return dbuser.User{}, errors.New("Email taken.")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	if err != nil {
		return dbuser.User{}, errors.New("Server error.")
	}

	user.Email = email
	user.Password = hash
	return user, nil
}
