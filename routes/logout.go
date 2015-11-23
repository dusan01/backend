package routes

import (
	"net/http"
	"time"
)

func logoutHandler(res http.ResponseWriter, req *http.Request) {
	http.SetCookie(res, &http.Cookie{
		Name:     "auth",
		Value:    "",
		Path:     "/",
		Domain:   "." + domain,
		Expires:  time.Now(),
		Secure:   !insecure,
		HttpOnly: !insecure,
		MaxAge:   -1,
	})

	http.Redirect(res, req, "/", 301)
}
