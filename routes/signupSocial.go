package routes

import (
	"encoding/json"
	"hybris/atlas"
	"hybris/db/dbsession"
	"hybris/enums"
	"net/http"
)

func signupSocialHandler(res http.ResponseWriter, req *http.Request) {
	var data struct {
		Username string `json:"username"`
		Token    string `json:"token"`
	}

	if err := json.NewDecoder(req.Body).Decode(&data); err != nil {
		WriteResponse(res, Response{enums.ResponseCodes.ServerError, "Server error.", nil})
		return
	}

	user, err := atlas.NewSocialUser(data.Username, data.Token)
	if err != nil {
		WriteResponse(res, Response{enums.ResponseCodes.BadRequest, err.Error(), nil})
		return
	}

	session, err := dbsession.New(user.Id)
	if err != nil {
		WriteResponse(res, Response{enums.ResponseCodes.ServerError, "Server error.", nil})
		return
	}

	if err := user.Save(); err != nil {
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
