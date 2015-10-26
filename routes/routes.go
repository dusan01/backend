package routes

import (
  "encoding/json"
  "github.com/gorilla/pat"
  "github.com/markbates/goth"
  "github.com/markbates/goth/gothic"
  "github.com/markbates/goth/providers/facebook"
  "github.com/markbates/goth/providers/twitter"
  "golang.org/x/crypto/bcrypt"
  "gopkg.in/mgo.v2/bson"
  "hybris/atlas"
  "hybris/db"
  "hybris/enums"
  "hybris/socket"
  "net/http"
  "net/url"
  "time"
)

func init() {
  goth.UseProviders(
    twitter.New("sVHYAm8YdmTn8H5R4zbqQ15db", "T80kt2I0n7fAJyMtihdn2zh0KCCbyYoUPpbbAJGTBIGp3q2Yir", "https://rglkjbfgd.turn.fm/_/auth/twitter/callback"),
    facebook.New("1626304387621454", "3d4bf252b325afda0ccf1c66af79ca98", "https://rglkjbfgd.turn.fm/_/auth/facebook/callback"),
  )
  gothic.GetState = func(req *http.Request) string {
    return req.URL.Query().Get("state")
  }
}

func Attach(router *pat.Router) {
  router.Get("/_/auth/{provider}/callback", authHandler)
  router.Get("/_/auth/{provider}", gothic.BeginAuthHandler)
  router.Post("/_/signup/social", signupSocialHandler)
  router.Post("/_/signup", signupHanlder)
  router.Post("/_/login", loginHanlder)
  router.Get("/_/socket", socketHandler)
  router.Get("/", indexHandler)
}

// Helper methods for writing
type Response struct {
  Status int         `json:"status"`
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
  case enums.RESPONSE_CODES.UNAUTHORIZED:
    res.WriteHeader(403)
  case enums.RESPONSE_CODES.ERROR:
    res.WriteHeader(500)
  }
  res.Write(data)
}

// Route handlers

func indexHandler(res http.ResponseWriter, req *http.Request) {
  http.Redirect(res, req, "https://turn.fm/", 301)
}

func authHandler(res http.ResponseWriter, req *http.Request) {
  info, err := gothic.CompleteUserAuth(res, req)
  if err != nil {
    WriteResponse(res, Response{enums.RESPONSE_CODES.BAD_REQUEST, nil})
    return
  }
  // Find the user and log them in if they exist
  accessToken := info.AccessToken + info.AccessTokenSecret

  query := bson.M{}
  query[info.Provider+"Token"] = accessToken

  user, err := db.GetUser(query)
  if err == nil {
    session, err := db.GetSession(bson.M{"userId": user.Id})
    if err != nil {
      session, err = db.NewSession(user.Id)
      if err != nil {
        WriteResponse(res, Response{enums.RESPONSE_CODES.ERROR, nil})
        return
      }
    }

    http.SetCookie(res, &http.Cookie{
      Name:     "auth",
      Value:    session.Cookie,
      Path:     "/",
      Domain:   ".turn.fm",
      Expires:  time.Now().Add(365 * 24 * time.Hour),
      Secure:   true,
      HttpOnly: true,
    })
    return
  }

  token := atlas.NewToken(info.Provider, accessToken)
  http.Redirect(res, req, "/signup/social?token="+url.QueryEscape(token), 301)
}

func signupSocialHandler(res http.ResponseWriter, req *http.Request) {
  var data struct {
    Username string `json:"username"`
    Token    string `json:"token"`
  }

  if err := json.NewDecoder(req.Body).Decode(&data); err != nil {
    WriteResponse(res, Response{enums.RESPONSE_CODES.BAD_REQUEST, nil})
    return
  }

  user, err := atlas.NewSocialUser(data.Username, data.Token)
  if err != nil {
    WriteResponse(res, Response{enums.RESPONSE_CODES.BAD_REQUEST, nil})
    return
  }

  // Create session
  session, err := db.NewSession(user.Id)
  if err != nil {
    WriteResponse(res, Response{enums.RESPONSE_CODES.ERROR, nil})
    return
  }

  // Save user
  if err := user.Save(); err != nil {
    WriteResponse(res, Response{enums.RESPONSE_CODES.ERROR, nil})
    return
  }

  // Save session
  if err := session.Save(); err != nil {
    WriteResponse(res, Response{enums.RESPONSE_CODES.ERROR, nil})
    return
  }

  http.SetCookie(res, &http.Cookie{
    Name:     "auth",
    Value:    session.Cookie,
    Path:     "/",
    Domain:   ".turn.fm",
    Expires:  time.Now().Add(365 * 24 * time.Hour),
    Secure:   true,
    HttpOnly: true,
  })

  WriteResponse(res, Response{enums.RESPONSE_CODES.OK, user.Struct()})
}

func signupHanlder(res http.ResponseWriter, req *http.Request) {
  var data struct {
    Username  string `json:"username"`
    Email     string `json:"email"`
    Password  string `json:"password"`
    Recaptcha string `json:"recaptcha"`
  }

  if err := json.NewDecoder(req.Body).Decode(&data); err != nil {
    WriteResponse(res, Response{enums.RESPONSE_CODES.BAD_REQUEST, nil})
    return
  }

  // ReCaptcha
  captchaClient := &http.Client{}
  captchaRes, err := captchaClient.PostForm("https://www.google.com/recaptcha/api/siteverify", url.Values{
    "secret":   {"6LdzaA8TAAAAAE7puUC6qhn2b2in89iiPL9s8_Nv"},
    "response": {data.Recaptcha},
  })
  if err != nil {
    WriteResponse(res, Response{enums.RESPONSE_CODES.ERROR, nil})
    return
  }

  var recaptchaData struct {
    Success bool `json:"success"`
  }

  if err := json.NewDecoder(captchaRes.Body).Decode(&recaptchaData); err != nil {
    WriteResponse(res, Response{enums.RESPONSE_CODES.ERROR, nil})
    return
  }

  if !recaptchaData.Success {
    WriteResponse(res, Response{enums.RESPONSE_CODES.BAD_REQUEST, nil})
    return
  }

  // Create User
  user, err := atlas.NewEmailUser(data.Username, data.Email, data.Password)
  if err != nil {
    WriteResponse(res, Response{enums.RESPONSE_CODES.BAD_REQUEST, nil})
    return
  }

  // Create session
  session, err := db.NewSession(user.Id)
  if err != nil {
    WriteResponse(res, Response{enums.RESPONSE_CODES.ERROR, nil})
    return
  }

  // Save user
  if err := user.Save(); err != nil {
    WriteResponse(res, Response{enums.RESPONSE_CODES.ERROR, nil})
    return
  }

  // Save session
  if err := session.Save(); err != nil {
    WriteResponse(res, Response{enums.RESPONSE_CODES.ERROR, nil})
    return
  }

  http.SetCookie(res, &http.Cookie{
    Name:     "auth",
    Value:    session.Cookie,
    Path:     "/",
    Domain:   ".turn.fm",
    Expires:  time.Now().Add(365 * 24 * time.Hour),
    Secure:   true,
    HttpOnly: true,
  })

  WriteResponse(res, Response{enums.RESPONSE_CODES.OK, user.Struct()})
}

func loginHanlder(res http.ResponseWriter, req *http.Request) {
  var data struct {
    Email    string `json:"email"`
    Password string `json:"password"`
  }

  if err := json.NewDecoder(req.Body).Decode(&data); err != nil {
    WriteResponse(res, Response{enums.RESPONSE_CODES.BAD_REQUEST, nil})
    return
  }

  // Get and authenticate user
  user, err := db.GetUser(bson.M{"email": data.Email})
  if err != nil {
    WriteResponse(res, Response{enums.RESPONSE_CODES.BAD_REQUEST, nil})
    return
  }

  if err := bcrypt.CompareHashAndPassword(user.Password, []byte(data.Password)); err != nil {
    WriteResponse(res, Response{enums.RESPONSE_CODES.UNAUTHORIZED, nil})
    return
  }

  // Get session
  session, err := db.GetSession(bson.M{"userId": user.Id})
  if err != nil {
    session, err = db.NewSession(user.Id)
    if err != nil {
      WriteResponse(res, Response{enums.RESPONSE_CODES.ERROR, nil})
      return
    }
  }

  http.SetCookie(res, &http.Cookie{
    Name:     "auth",
    Value:    session.Cookie,
    Path:     "/",
    Domain:   ".turn.fm",
    Expires:  time.Now().Add(365 * 24 * time.Hour),
    Secure:   true,
    HttpOnly: true,
  })

  WriteResponse(res, Response{enums.RESPONSE_CODES.OK, user.Struct()})
}

func socketHandler(res http.ResponseWriter, req *http.Request) {
  socket.NewSocket(res, req)
}
