package socket

import (
  "encoding/json"
  "github.com/gorilla/websocket"
  "hybris/debug"
  "hybris/pool"
  "net/http"
  "strings"
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
    debug.Log("[socket/NewSocket] Failed to upgrade socket: [%s]", err.Error())
    return
  }

  conn.SetReadLimit(4096)
  conn.SetReadDeadline(time.Now().Add(PongWait))
  conn.SetPongHandler(func(string) error {
    conn.SetReadDeadline(time.Now().Add(PongWait))
    return nil
  })

  go Heartbeat(conn)

  dcTimer := time.AfterFunc(time.Second*10, func() {
    conn.Close()
  })

  if _, msg, err := conn.ReadMessage(); err == nil {
    var data struct {
      Hello  bool   `json:"hello"`
      Server string `json:"frontend-auth"`
    }

    if err := json.Unmarshal(msg, &data); err != nil {
      conn.Close()
      return
    }

    if data.Hello {
      dcTimer.Stop()
      debug.Log("[socket/NewSocket] Client handshake successful")
      pool.NewClient(req, conn)
    } else if data.Server == "09fj032jf093j09mVJVWOimjzoimvor3imjmR23v43" && strings.Split(req.RemoteAddr, ":")[0] == "127.0.0.1" {
      dcTimer.Stop()
      pool.NewServer(conn)
    } else {
      debug.Log("[socket/NewSocket] Received bad handshake")
      conn.Close()
    }
  } else {
    debug.Log("[socket/NewSocket] Connection terminated unexpectedly. Reason: %s", err.Error())
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
