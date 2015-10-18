package routes

import (
  "encoding/json"
  "github.com/gorilla/mux"
  "golang.org/x/crypto/bcrypt"
  "gopkg.in/mgo.v2/bson"
  "hybris/db"
  "hybris/socket"
  "net/http"
)

func Attach(router *mux.Router) {
  router.HandleFunc("/", indexHandler)
  router.HandleFunc("/_/signup", signupHanlder).Methods("POST")
  router.HandleFunc("/_/login", loginHanlder).Methods("POST")
  router.HandleFunc("/_/socket", socketHandler)
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
  case 0:
    res.WriteHeader(200)
  case 1:
    res.WriteHeader(400)
  case 2:
    res.WriteHeader(403)
  case 3:
    res.WriteHeader(500)
  }
  res.Write(data)
}

// Route handlers

func indexHandler(res http.ResponseWriter, req *http.Request) {
  http.Redirect(res, req, "https://turn.fm/", 301)
}

func signupHanlder(res http.ResponseWriter, req *http.Request) {
  var data struct {
    Username string `json:"username"`
    Email    string `json:"email"`
    Password string `json:"password"`
  }

  decoder := json.NewDecoder(req.Body)
  if err := decoder.Decode(&data); err != nil {
    WriteResponse(res, Response{1, nil})
    return
  }

  // Create User
  user, err := db.NewUser(data.Username, data.Email, data.Password)
  if err != nil {
    WriteResponse(res, Response{1, nil})
    return
  }

  // Create session
  session, err := db.NewSession(user.Id)
  if err != nil {
    WriteResponse(res, Response{3, nil})
    return
  }

  // Save user
  if err := user.Save(); err != nil {
    WriteResponse(res, Response{3, nil})
    return
  }

  // Save session
  if err := session.Save(); err != nil {
    WriteResponse(res, Response{3, nil})
    return
  }

  http.SetCookie(res, &http.Cookie{
    Name:     "auth",
    Value:    session.Cookie,
    Domain:   ".turn.fm",
    Secure:   true,
    HttpOnly: true,
  })

  WriteResponse(res, Response{0, user.Struct()})
}

func loginHanlder(res http.ResponseWriter, req *http.Request) {
  var data struct {
    Email    string `json:"email"`
    Password string `json:"password"`
  }

  decoder := json.NewDecoder(req.Body)
  if err := decoder.Decode(&data); err != nil {
    WriteResponse(res, Response{1, nil})
    return
  }

  // Get and authenticate user
  user, err := db.GetUser(bson.M{"email": data.Email})
  if err != nil {
    WriteResponse(res, Response{1, nil})
    return
  }

  if err := bcrypt.CompareHashAndPassword(user.Password, []byte(data.Password)); err != nil {
    WriteResponse(res, Response{2, nil})
    return
  }

  // Get session
  session, err := db.GetSession(bson.M{"userid": user.Id})
  if err != nil {
    session, err = db.NewSession(user.Id)
    if err != nil {
      WriteResponse(res, Response{3, nil})
      return
    }
  }

  http.SetCookie(res, &http.Cookie{
    Name:     "auth",
    Value:    session.Cookie,
    Domain:   ".turn.fm",
    Secure:   true,
    HttpOnly: true,
  })

  WriteResponse(res, Response{0, user.Struct()})
}

func socketHandler(res http.ResponseWriter, req *http.Request) {
  socket.NewSocket(res, req)
}
