package socket

import (
  "encoding/json"
  "errors"
  "github.com/gorilla/websocket"
  "hybris/socket/client"
  "hybris/socket/frontend"
  "net/http"
  "strings"
  "time"
)

const (
  readLimit         = 4096
  pongWait          = 55 * time.Second
  disconnectTimeout = 10 * time.Second
)

var upgrader = websocket.Upgrader{
  ReadBufferSize:  1024,
  WriteBufferSize: 1024,
  CheckOrigin: func(r *http.Request) bool {
    return true
  },
}

type Socket interface {
  Send([]byte)
  Terminate()
}

func New(res http.ResponseWriter, req *http.Request) (Socket, error) {
  conn, err := upgrader.Upgrade(res, req, nil)
  if err != nil {
    return nil, errors.New("Could not upgrade connection")
  }

  conn.SetReadLimit(readLimit)
  conn.SetReadDeadline(time.Now().Add(pongWait))
  conn.SetPongHandler(func(string) error {
    conn.SetReadDeadline(time.Now().Add(pongWait))
    return nil
  })

  disconnectTimer := time.AfterFunc(disconnectTimeout, func() {
    conn.Close()
  })

  _, msg, err := conn.ReadMessage()
  if err != nil {
    return nil, errors.New("Failed to read message")
  }

  disconnectTimer.Stop()

  var data struct {
    Hello        bool   `json:"hello"`
    FrontendAuth string `json:"frontend-auth"`
  }

  if err := json.Unmarshal(msg, &data); err != nil {
    return nil, errors.New("Failed to unmarshal message")
  }

  switch {
  case data.Hello:
    return client.New(req, conn)
  case data.FrontendAuth == "09fj032jf093j09mVJVWOimjzoimvor3imjmR23v43" && strings.Split(req.RemoteAddr, ":")[0] == "127.0.0.1":
    return frontend.New(req, conn)
  }
  return nil, errors.New("Invalid message received")
}
