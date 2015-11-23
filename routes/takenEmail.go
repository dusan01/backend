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

func takenEmailHandler(res http.ResponseWriter, req *http.Request) {
	email := strings.ToLower(strings.TrimSpace(req.URL.Query().Get(":email")))

	if !validation.Email(email) {
		WriteResponse(res, Response{enums.ResponseCodes.BadRequest, "Invalid email.", nil})
		return
	}

	if _, err := dbuser.Get(uppdb.Cond{"email": email}); err == uppdb.ErrNoMoreRows {
		WriteResponse(res, Response{enums.ResponseCodes.Ok, "", message.S{"taken": false}})
		return
	} else if err != nil {
		WriteResponse(res, Response{enums.ResponseCodes.ServerError, "Server error.", nil})
		return
	}

	WriteResponse(res, Response{enums.ResponseCodes.Ok, "", message.S{"taken": true}})
}
