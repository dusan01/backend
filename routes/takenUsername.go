package routes

import (
	"hybris/db/dbuser"
	"hybris/enums"
	"hybris/socket/message"
	"hybris/validation"
	"net/http"
	"strings"

	uppdb "upper.io/db"
)

func takenUsernameHandler(res http.ResponseWriter, req *http.Request) {
	username := strings.ToLower(strings.TrimSpace(req.URL.Query().Get(":username")))

	if !validation.Username(username) {
		WriteResponse(res, Response{enums.ResponseCodes.BadRequest, "Invalid username.", nil})
		return
	}

	if _, err := dbuser.Get(uppdb.Cond{"username": username}); err == uppdb.ErrNoMoreRows {
		WriteResponse(res, Response{enums.ResponseCodes.Ok, "", message.S{"taken": false}})
		return
	} else if err != nil {
		WriteResponse(res, Response{enums.ResponseCodes.ServerError, "Server error.", nil})
		return
	}

	WriteResponse(res, Response{enums.ResponseCodes.Ok, "", message.S{"taken": true}})
}
