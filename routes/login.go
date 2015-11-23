package routes

import (
	"encoding/json"
	"hybris/db/dbsession"
	"hybris/db/dbuser"
	"hybris/enums"
	"net/http"

	"golang.org/x/crypto/bcrypt"

	uppdb "upper.io/db"
)

func loginHandler(res http.ResponseWriter, req *http.Request) {
	var data struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(req.Body).Decode(&data); err != nil {
		WriteResponse(res, Response{enums.ResponseCodes.ServerError, "Server error.", nil})
		return
	}

	user, err := dbuser.Get(uppdb.Cond{"email": data.Email})
	if err == uppdb.ErrNoMoreRows {
		WriteResponse(res, Response{enums.ResponseCodes.Forbidden, "Wrong email or password.", nil})
		return
	} else if err != nil {
		WriteResponse(res, Response{enums.ResponseCodes.ServerError, "Server error.", nil})
		return
	}

	if err := bcrypt.CompareHashAndPassword(user.Password, []byte(data.Password)); err != nil {
		WriteResponse(res, Response{enums.ResponseCodes.Forbidden, "Wrong email or password.", nil})
		return
	}

	session, err := dbsession.New(user.Id)
	if err != nil {
		WriteResponse(res, Response{enums.ResponseCodes.ServerError, "Server error.", nil})
		return
	}

	if err := session.Save(); err != nil {
		WriteResponse(res, Response{enums.ResponseCodes.ServerError, "Server error.", nil})
		return
	}

	SetCookie(res, session.Cookie)
	WriteResponse(res, Response{enums.ResponseCodes.Ok, "", user.Struct()})
}
