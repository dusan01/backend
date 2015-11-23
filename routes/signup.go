package routes

import (
	"encoding/json"
	"fmt"
	"hybris/atlas"
	"hybris/db/dbsession"
	"hybris/enums"
	"net/http"
	"net/url"
	"strings"
)

func signupHandler(res http.ResponseWriter, req *http.Request) {
	var data struct {
		Username  string `json:"username"`
		Email     string `json:"email"`
		Password  string `json:"password"`
		Recaptcha string `json:"recaptcha"`
	}

	if err := json.NewDecoder(req.Body).Decode(&data); err != nil {
		WriteResponse(res, Response{enums.ResponseCodes.ServerError, "Server error.", nil})
		return
	}

	if !nocaptcha {
		captchaClient := &http.Client{}
		captchaRes, err := captchaClient.PostForm("https://www.google.com/recaptcha/api/siteverify", url.Values{
			"secret":   {"6LfDhg4TAAAAALGzHUmWr-zcuNVgE5oU2PYjVj4I"},
			"response": {data.Recaptcha},
			"remoteip": {strings.Split(req.RemoteAddr, ":")[0]},
		})
		if err != nil {
			WriteResponse(res, Response{enums.ResponseCodes.ServerError, "Server error.", nil})
			return
		}

		var recaptchaData struct {
			Success bool `json:"success"`
		}

		if err := json.NewDecoder(captchaRes.Body).Decode(&recaptchaData); err != nil {
			WriteResponse(res, Response{enums.ResponseCodes.ServerError, "Server error.", nil})
			return
		}

		if !recaptchaData.Success {
			WriteResponse(res, Response{enums.ResponseCodes.BadRequest, "Invalid recaptcha.", nil})
			return
		}
	}

	user, err := atlas.NewEmailUser(data.Username, data.Email, data.Password)
	if err != nil {
		WriteResponse(res, Response{enums.ResponseCodes.BadRequest, err.Error(), nil})
		return
	}

	session, err := dbsession.New(user.Id)
	if err != nil {
		WriteResponse(res, Response{enums.ResponseCodes.ServerError, "Server error.", nil})
		return
	}

	fmt.Println(user.Id)

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
