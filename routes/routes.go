package routes

import (
  "encoding/json"
  "fmt"
  "github.com/gorilla/pat"
  "github.com/gorilla/securecookie"
  "github.com/gorilla/sessions"
  "github.com/markbates/goth"
  "github.com/markbates/goth/gothic"
  "github.com/markbates/goth/providers/facebook"
  "github.com/markbates/goth/providers/twitter"
  "golang.org/x/crypto/bcrypt"
  "gopkg.in/mgo.v2"
  "gopkg.in/mgo.v2/bson"
  "hybris/atlas"
  "hybris/db"
  "hybris/enums"
  "hybris/socket"
  "hybris/validation"
  "net/http"
  "net/url"
  "strings"
  "time"
)

func init() {
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
  router.Post("/signup", signupHanlder)
  router.Post("/login", loginHanlder)
  router.Get("/logout", logoutHandler)
  router.Get("/taken/username/{username}", takenUsernameHandler)
  router.Get("/taken/email/{email}", takenEmailHandler)
  router.Get("/socket", socketHandler)
  router.Get("/", indexHandler)
}

// Helper methods for writing
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
  case enums.RESPONSE_CODES.OK:
    res.WriteHeader(200)
  case enums.RESPONSE_CODES.BAD_REQUEST:
    res.WriteHeader(400)
  case enums.RESPONSE_CODES.FORBIDDEN:
    res.WriteHeader(403)
  case enums.RESPONSE_CODES.SERVER_ERROR:
    res.WriteHeader(500)
  }
  res.Write(data)
}

// Route handlers

func indexHandler(res http.ResponseWriter, req *http.Request) {
  res.Write([]byte(`turnf.fm backend, version ` + enums.VERSION))
}

func authHandler(res http.ResponseWriter, req *http.Request) {
  accessToken, query, token, loggedIn, failed := "", bson.M{}, "", false, false

  info, err := gothic.CompleteUserAuth(res, req)
  if err != nil {
    failed = true
    goto writeSocial
  }
  // Find the user and log them in if they exist
  accessToken = info.AccessToken + info.AccessTokenSecret
  query[info.Provider+"Token"] = accessToken
  if user, err := db.GetUser(query); err == nil {
    session, err := db.NewSession(user.Id)
    if err != nil {
      failed = true
      goto writeSocial
    }

    if err := session.Save(); err != nil {
      failed = true
      goto writeSocial
    }

    http.SetCookie(res, &http.Cookie{
      Name:     "auth",
      Value:    session.Cookie,
      Path:     "/",
      Domain:   ".turn.fm",
      Expires:  time.Now().Add(365 * 24 * time.Hour),
      Secure:   false,
      HttpOnly: false,
    })
    loggedIn = true
  } else {
    token = atlas.NewToken(info.Provider, accessToken)
  }

writeSocial:
  res.Header().Set("Content-Type", "text/html; encoding=utf-8")
  res.Write([]byte(fmt.Sprintf(`
    <!doctype html>
    <html>
    <head>
      <title>Callback</title>
    </head>
    <body style="background: #1A2326;color: white; font-family: sans-serif;">
      <div style="position: absolute;top:50%;left:50%; transform: translate(-50%, -50%);">This window should close automatically.</div>
      <script>
        window.opener.setTimeout(function() {
          window.opener.TURN_SOCIAL_CALLBACK({
            token: '%s',
            type: '%s',
            loggedIn: %t,
            failed: %t
          });
        }, 1);
      </script>
    </body>
  </html>
  `, token, info.Provider, loggedIn, failed)))
}

func signupSocialHandler(res http.ResponseWriter, req *http.Request) {
  var data struct {
    Username string `json:"username"`
    Token    string `json:"token"`
  }

  if err := json.NewDecoder(req.Body).Decode(&data); err != nil {
    WriteResponse(res, Response{enums.RESPONSE_CODES.SERVER_ERROR, "Server error.", nil})
    return
  }

  user, err := atlas.NewSocialUser(data.Username, data.Token)
  if err != nil {
    WriteResponse(res, Response{enums.RESPONSE_CODES.BAD_REQUEST, err.Error(), nil})
    return
  }

  // Create session
  session, err := db.NewSession(user.Id)
  if err != nil {
    WriteResponse(res, Response{enums.RESPONSE_CODES.SERVER_ERROR, "Server error.", nil})
    return
  }

  // Save user
  if err := user.Save(); err != nil {
    WriteResponse(res, Response{enums.RESPONSE_CODES.SERVER_ERROR, "Server error.", nil})
    return
  }

  // Save session
  if err := session.Save(); err != nil {
    WriteResponse(res, Response{enums.RESPONSE_CODES.SERVER_ERROR, "Server error.", nil})
    return
  }

  http.SetCookie(res, &http.Cookie{
    Name:     "auth",
    Value:    session.Cookie,
    Path:     "/",
    Domain:   ".turn.fm",
    Expires:  time.Now().Add(365 * 24 * time.Hour),
    Secure:   false,
    HttpOnly: false,
  })

  WriteResponse(res, Response{enums.RESPONSE_CODES.OK, "", user.Struct()})
}

func signupHanlder(res http.ResponseWriter, req *http.Request) {
  var data struct {
    Username  string `json:"username"`
    Email     string `json:"email"`
    Password  string `json:"password"`
    Recaptcha string `json:"recaptcha"`
  }

  if err := json.NewDecoder(req.Body).Decode(&data); err != nil {
    WriteResponse(res, Response{enums.RESPONSE_CODES.SERVER_ERROR, "Server error.", nil})
    return
  }

  // ReCaptcha
  captchaClient := &http.Client{}
  captchaRes, err := captchaClient.PostForm("https://www.google.com/recaptcha/api/siteverify", url.Values{
    "secret":   {"6LfDhg4TAAAAALGzHUmWr-zcuNVgE5oU2PYjVj4I"},
    "response": {data.Recaptcha},
    "remoteip": {strings.Split(req.RemoteAddr, ":")[0]},
  })
  if err != nil {
    WriteResponse(res, Response{enums.RESPONSE_CODES.SERVER_ERROR, "Server error.", nil})
    return
  }

  var recaptchaData struct {
    Success bool `json:"success"`
  }

  if err := json.NewDecoder(captchaRes.Body).Decode(&recaptchaData); err != nil {
    WriteResponse(res, Response{enums.RESPONSE_CODES.SERVER_ERROR, "Server error.", nil})
    return
  }

  if !recaptchaData.Success {
    WriteResponse(res, Response{enums.RESPONSE_CODES.BAD_REQUEST, "Invalid recaptcha.", nil})
    return
  }

  // Create User
  user, err := atlas.NewEmailUser(data.Username, data.Email, data.Password)
  if err != nil {
    WriteResponse(res, Response{enums.RESPONSE_CODES.BAD_REQUEST, err.Error(), nil})
    return
  }

  // Create session
  session, err := db.NewSession(user.Id)
  if err != nil {
    WriteResponse(res, Response{enums.RESPONSE_CODES.SERVER_ERROR, "Server error.", nil})
    return
  }

  // Save user
  if err := user.Save(); err != nil {
    WriteResponse(res, Response{enums.RESPONSE_CODES.SERVER_ERROR, "Server error.", nil})
    return
  }

  // Save session
  if err := session.Save(); err != nil {
    WriteResponse(res, Response{enums.RESPONSE_CODES.SERVER_ERROR, "Server error.", nil})
    return
  }

  http.SetCookie(res, &http.Cookie{
    Name:     "auth",
    Value:    session.Cookie,
    Path:     "/",
    Domain:   ".turn.fm",
    Expires:  time.Now().Add(365 * 24 * time.Hour),
    Secure:   false,
    HttpOnly: false,
  })

  WriteResponse(res, Response{enums.RESPONSE_CODES.OK, "", user.Struct()})
}

func loginHanlder(res http.ResponseWriter, req *http.Request) {
  var data struct {
    Email    string `json:"email"`
    Password string `json:"password"`
  }

  if err := json.NewDecoder(req.Body).Decode(&data); err != nil {
    WriteResponse(res, Response{enums.RESPONSE_CODES.SERVER_ERROR, "Server error.", nil})
    return
  }

  // Get and authenticate user
  user, err := db.GetUser(bson.M{"email": data.Email})
  if err == mgo.ErrNotFound {
    WriteResponse(res, Response{enums.RESPONSE_CODES.FORBIDDEN, "Wrong email or password.", nil})
    return
  } else if err != nil {
    WriteResponse(res, Response{enums.RESPONSE_CODES.SERVER_ERROR, "Server error.", nil})
    return
  }

  if err := bcrypt.CompareHashAndPassword(user.Password, []byte(data.Password)); err != nil {
    WriteResponse(res, Response{enums.RESPONSE_CODES.FORBIDDEN, "Wrong email or password.", nil})
    return
  }

  // Get session
  session, err := db.NewSession(user.Id)
  if err != nil {
    WriteResponse(res, Response{enums.RESPONSE_CODES.SERVER_ERROR, "Server error.", nil})
    return
  }

  if err := session.Save(); err != nil {
    WriteResponse(res, Response{enums.RESPONSE_CODES.SERVER_ERROR, "Server error.", nil})
    return
  }

  http.SetCookie(res, &http.Cookie{
    Name:     "auth",
    Value:    session.Cookie,
    Path:     "/",
    Domain:   ".turn.fm",
    Expires:  time.Now().Add(365 * 24 * time.Hour),
    Secure:   false,
    HttpOnly: false,
  })

  WriteResponse(res, Response{enums.RESPONSE_CODES.OK, "", user.Struct()})
}

func logoutHandler(res http.ResponseWriter, req *http.Request) {
  http.SetCookie(res, &http.Cookie{
    Name:     "auth",
    Value:    "",
    Path:     "/",
    Domain:   ".turn.fm",
    Expires:  time.Now(),
    Secure:   false,
    HttpOnly: false,
  })

  http.Redirect(res, req, "/", 301)
}

func takenUsernameHandler(res http.ResponseWriter, req *http.Request) {
  username := strings.TrimSpace(req.URL.Query().Get(":username"))

  if !validation.Username(username) {
    WriteResponse(res, Response{enums.RESPONSE_CODES.BAD_REQUEST, "Invalid username.", nil})
    return
  }

  if _, err := db.GetUser(bson.M{"username": username}); err == mgo.ErrNotFound {
    WriteResponse(res, Response{enums.RESPONSE_CODES.OK, "", map[string]interface{}{
      "taken": false,
    }})
    return
  } else if err != nil {
    WriteResponse(res, Response{enums.RESPONSE_CODES.SERVER_ERROR, "Server error.", nil})
    return
  }

  WriteResponse(res, Response{enums.RESPONSE_CODES.OK, "", map[string]interface{}{
    "taken": true,
  }})
}

func takenEmailHandler(res http.ResponseWriter, req *http.Request) {
  email := strings.TrimSpace(req.URL.Query().Get(":email"))

  if !validation.Email(email) {
    WriteResponse(res, Response{enums.RESPONSE_CODES.BAD_REQUEST, "Invalid email.", nil})
    return
  }

  if _, err := db.GetUser(bson.M{"email": email}); err == mgo.ErrNotFound {
    WriteResponse(res, Response{enums.RESPONSE_CODES.OK, "", map[string]interface{}{
      "taken": false,
    }})
    return
  } else if err != nil {
    WriteResponse(res, Response{enums.RESPONSE_CODES.SERVER_ERROR, "Server error.", nil})
    return
  }

  WriteResponse(res, Response{enums.RESPONSE_CODES.OK, "", map[string]interface{}{
    "taken": true,
  }})
}

func socketHandler(res http.ResponseWriter, req *http.Request) {
  socket.NewSocket(res, req)
}
