package atlas

import (
	"errors"
	"hybris/db/dbuser"
	"hybris/debug"
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
	debug.Log("Generating new token")
	for k, v := range sessions {
		if v.UserId == userId {
			debug.Log("Token %s already exists", k)
			token = k
			return
		}
	}
	token = strings.Replace(uuid.NewUUID().String(), "-", "", -1)
	sessions[token] = Session{
		provider,
		userId,
	}
	debug.Log("Genetated token %s", token)
	return
}

func AddIntegration(user *dbuser.User, token string) error {
	debug.Log("Adding integrating into user")
	session, ok := sessions[token]
	if !ok {
		debug.Log("Invalid token for integration")
		return errors.New("Invalid token.")
	}

	switch session.Provider {
	case "facebook":
		debug.Log("Adding integration for facebook")
		user.FacebookId = session.UserId
	case "twitter":
		debug.Log("Adding integration for twitter")
		user.TwitterId = session.UserId
	}

	debug.Log("Deleting session %s", token)
	delete(sessions, token)

	return nil
}

func NewSocialUser(username, token string) (dbuser.User, error) {
	debug.Log("Creating new user using social parameters")
	user, err := dbuser.New(username)
	if err != nil {
		debug.Log("Could not create new social user: %s", err.Error())
		return dbuser.User{}, err
	}
	if err := AddIntegration(&user, token); err != nil {
		debug.Log("Could not add integration: %s", err.Error())
		return dbuser.User{}, err
	}

	debug.Log("Successfully created social user")
	return user, nil
}

func NewEmailUser(username, email, password string) (dbuser.User, error) {
	debug.Log("Creating new user with email")
	user, err := dbuser.New(username)
	if err != nil {
		debug.Log("Could not create new email user: %s", err.Error())
		return dbuser.User{}, err
	}

	email = strings.ToLower(email)
	if !validation.Email(email) {
		debug.Log("Email is invalid")
		return dbuser.User{}, errors.New("Invalid email.")
	}
	if !validation.Password(password) {
		debug.Log("Password is invalid")
		return dbuser.User{}, errors.New("Invalid password.")
	}
	if _, err := dbuser.Get(uppdb.Cond{"email": email}); err == nil {
		debug.Log("Email is alredy in use")
		return dbuser.User{}, errors.New("Email taken.")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	if err != nil {
		debug.Log("Could not hash password: %s", err.Error())
		return dbuser.User{}, errors.New("Server error.")
	}

	debug.Log("Successfully created new email user")

	user.Email = email
	user.Password = hash
	return user, nil
}
