package pool

import (
  "encoding/json"
  "github.com/gorilla/websocket"
  "gopkg.in/mgo.v2/bson"
  "hybris/db"
  "hybris/debug"
  "hybris/enums"
  "sync"
  "time"
)

type Server struct {
  sync.Mutex
  Conn  *websocket.Conn
  ConnM sync.Mutex
}

func NewServer(conn *websocket.Conn) {
  server := &Server{
    Conn: conn,
  }

  server.Send([]byte(`{"__auth": true}`))
  go server.Listen()
  debug.Log("[pool/NewServer] Successfully connected internal server")
}

func (s *Server) Terminate() {
  s.Conn.Close()
}

func (s *Server) Listen() {
  defer s.Terminate()
  for {
    if _, msg, err := s.Conn.ReadMessage(); err == nil {
      go s.Receive(msg)
    } else {
      return
    }
  }
}

func (s *Server) Send(data []byte) {
  s.ConnM.Lock()
  defer s.ConnM.Unlock()
  s.Conn.SetWriteDeadline(time.Now().Add(55 * time.Second))
  if err := s.Conn.WriteMessage(websocket.TextMessage, data); err != nil {
    debug.Log("[pool/Server.Send] Failed to send message: [%s]", err.Error())
    s.Terminate()
  }
}

func (s *Server) Receive(msg []byte) {
  var r struct {
    Id     string          `json:"i"`
    Action string          `json:"a"`
    Data   json.RawMessage `json:"d"`
  }

  if err := json.Unmarshal(msg, &r); err != nil {
    debug.Log("[pool/Server.Receive] Server sent bad data")
    return
  }

  switch r.Action {
  case "cook":
    var data struct {
      Ip   string `json:"ip"`
      Auth string `json:"auth"`
    }

    if err := json.Unmarshal(r.Data, &data); err != nil {
      debug.Log("[pool/Server.Receive -> cook] Failed to unmarshal json. Reason: %s", err.Error())
      NewAction(r.Id, enums.RESPONSE_CODES.BAD_REQUEST, r.Action, nil).Dispatch(s)
      return
    }

    _, err := db.GetSession(bson.M{"cookie": data.Auth})
    debug.Log("[pool/Server.Receive -> cook] Cookie exists: [%t]", err == nil)
    NewAction(r.Id, enums.RESPONSE_CODES.OK, r.Action, map[string]interface{}{
      "ok": err == nil,
    }).Dispatch(s)
  }
}
