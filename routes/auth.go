package routes

import (
	"html/template"
	"hybris/atlas"
	"hybris/db/dbsession"
	"hybris/db/dbuser"
	"net/http"

	"github.com/markbates/goth/gothic"
	uppdb "upper.io/db"
)

func authHandler(res http.ResponseWriter, req *http.Request) {
	token, provider, loggedIn, failed := "", "", false, false
	info, err := gothic.CompleteUserAuth(res, req)
	if err != nil {
		failed = true
		writeSocialWindowResponse(res, token, provider, loggedIn, failed)
		return
	}

	provider = info.Provider
	userId := info.UserID
	query := uppdb.Cond{}
	query[provider+"Id"] = userId
	user, err := dbuser.Get(query)
	if err == uppdb.ErrNoMoreRows {
		token = atlas.NewToken(info.Provider, userId)
		writeSocialWindowResponse(res, token, provider, loggedIn, failed)
		return
	} else if err != nil {
		failed = true
		writeSocialWindowResponse(res, token, provider, loggedIn, failed)
		return
	}

	session, err := dbsession.New(user.Id)
	if err != nil {
		failed = true
		writeSocialWindowResponse(res, token, provider, loggedIn, failed)
		return
	}

	if err := session.Save(); err != nil {
		failed = true
		writeSocialWindowResponse(res, token, provider, loggedIn, failed)
		return
	}

	loggedIn = true
	SetCookie(res, session.Cookie)
	writeSocialWindowResponse(res, token, provider, loggedIn, failed)
}

func writeSocialWindowResponse(res http.ResponseWriter, token, provider string, loggedIn, failed bool) {
	res.Header().Set("Content-Type", "text/html; encoding=utf-8")
	tmpl, err := template.New("test").Parse(`
        <!doctype html>
        <html>
        <head>
                <title>Callback</title>
        </head>
        <body style="background: #1A2326;color: white; font-family: sans-serif;">
                <div style="position: absolute; top:50%; left:50%; transform: translate(-50%, -50%);">This window should close automatically.</div>
                <script>
                        window.opener.setTimeout(function() {
                                window.opener.TURN_SOCIAL_CALLBACK({
                                        token: '{{.Token}}',
                                        type: '{{.Provider}}',
                                        loggedIn: {{.LoggedIn}},
                                        failed: {{.Failed}}
                                });
                        }, 1);
                        window.close()
                </script>
        </body>
        </html>
        `)
	if err != nil {
		res.WriteHeader(500)
	}

	if err := tmpl.Execute(res, struct {
		Token    string
		Provider string
		LoggedIn bool
		Failed   bool
	}{token, provider, loggedIn, failed}); err != nil {
		res.WriteHeader(500)
	}
}
