package routes

import (
	"encoding/json"
	"flag"
	"hybris/enums"
	"net/http"
	"time"

	"github.com/gorilla/pat"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/facebook"
	"github.com/markbates/goth/providers/twitter"
)

var (
	nocaptcha bool
	insecure  bool
	domain    string = "turn.fm"
)

func init() {
	flag.BoolVar(&nocaptcha, "nocaptcha", false, "Determines whether or not routes will use recaptcha")
	flag.BoolVar(&insecure, "insecure", false, "Whether cookies are secured or not")
	flag.StringVar(&domain, "domain", "domain.extension", "Domain override")

	gothic.Store = sessions.NewCookieStore(securecookie.GenerateRandomKey(64))
	goth.UseProviders(
		twitter.New("sVHYAm8YdmTn8H5R4zbqQ15db", "T80kt2I0n7fAJyMtihdn2zh0KCCbyYoUPpbbAJGTBIGp3q2Yir", "http://devv.turn.fm/_/auth/twitter/callback"),
		facebook.New("1626304387621454", "3d4bf252b325afda0ccf1c66af79ca98", "http://devv.turn.fm/_/auth/facebook/callback"),
	)
	gothic.GetState = func(req *http.Request) string {
		return req.URL.Query().Get("state")
	}
}

func Attach(router *pat.Router) {
	router.Get("/auth/{provider}/callback", authHandler)
	router.Get("/auth/{provider}", gothic.BeginAuthHandler)
	router.Post("/signup/social", signupSocialHandler)
	router.Post("/signup", signupHandler)
	router.Post("/login", loginHandler)
	router.Get("/logout", logoutHandler)
	router.Get("/taken/username/{username}", takenUsernameHandler)
	router.Get("/taken/email/{email}", takenEmailHandler)
	router.Get("/socket", socketHandler)
	router.Get("/", indexHandler)
}

type Response struct {
	Status int         `json:"status"`
	Reason string      `json:"reason,omitempty"`
	Data   interface{} `json:"data"`
}

func WriteResponse(res http.ResponseWriter, response Response) {
	data, err := json.Marshal(response)
	if err != nil {
		res.WriteHeader(500)
		return
	}

	res.Header().Set("Content-Type", "application/json; encoding=utf-8")
	switch response.Status {
	case enums.ResponseCodes.Ok:
		res.WriteHeader(200)
	case enums.ResponseCodes.BadRequest:
		res.WriteHeader(400)
	case enums.ResponseCodes.Forbidden:
		res.WriteHeader(403)
	case enums.ResponseCodes.ServerError:
		res.WriteHeader(500)
	}
	res.Write(data)
}

func SetCookie(res http.ResponseWriter, value string) {
	http.SetCookie(res, &http.Cookie{
		Name:     "auth",
		Value:    value,
		Path:     "/",
		Domain:   "." + domain,
		Expires:  time.Now().Add(365 * 24 * time.Hour),
		Secure:   !insecure,
		HttpOnly: !insecure,
	})
}
