package socket

import (
  "encoding/json"
  "github.com/gorilla/websocket"
  "hybris/debug"
  "hybris/pool"
  "net/http"
  "time"
)

const (
  PingPeriod time.Duration = 10 * time.Second
  PongWait   time.Duration = 55 * time.Second
  WriteWait  time.Duration = 55 * time.Second
)

var Upgrader = websocket.Upgrader{
  ReadBufferSize:  1024,
  WriteBufferSize: 1024,
  CheckOrigin: func(r *http.Request) bool {
    return true
  },
}

func NewSocket(res http.ResponseWriter, req *http.Request) {
  conn, err := Upgrader.Upgrade(res, req, nil)
  if err != nil {
    return
  }

  conn.SetReadLimit(4096)
  conn.SetReadDeadline(time.Now().Add(PongWait))
  conn.SetPongHandler(func(string) error {
    conn.SetReadDeadline(time.Now().Add(PongWait))
    return nil
  })

  go Heartbeat(conn)

  if _, msg, err := conn.ReadMessage(); err == nil {
    var data struct {
      Hello  bool   `json:"hello"`
      Server string `json:"---conn_294857"`
    }

    if err := json.Unmarshal(msg, &data); err != nil {
      conn.Close()
      return
    }

    if data.Hello {
      go debug.Log("Client handshake successful")
      pool.NewClient(req, conn)
    } else if data.Server == "" && req.Header.Get("X-Forwarded-For") == "127.0.0.1" {
      // pool.NewServer()
    } else {
      conn.Close()
    }
  } else {
    conn.Close()
  }
}

func Heartbeat(conn *websocket.Conn) {
  ticker := time.NewTicker(PingPeriod)
  defer ticker.Stop()
  for {
    <-ticker.C
    conn.SetWriteDeadline(time.Now().Add(WriteWait))
    if err := conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
      conn.Close()
      return
    }
  }
}
