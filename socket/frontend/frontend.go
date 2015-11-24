package frontend

import (
	"hybris/socket/frontend/frontendaction"
	"hybris/socket/message"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// move to a new hybris/constants package?
	writeTimeout = 55 * time.Second
	pingPeriod   = 10 * time.Second
)

type Frontend struct {
	sync.Mutex
	Conn  *websocket.Conn
	ConnM sync.Mutex
}

func New(req *http.Request, conn *websocket.Conn) (*Frontend, error) {
	f := &Frontend{
		Conn: conn,
	}

	message.NewUnique(message.S{
		"__auth": true,
	}).Dispatch(f)

	go f.heartbeat()
	go f.listen()
	return f, nil
}

func (f *Frontend) Send(data []byte) {
	f.ConnM.Lock()
	defer f.ConnM.Unlock()
	conn := f.Conn
	conn.SetWriteDeadline(time.Now().Add(writeTimeout))
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		f.Terminate()
	}
}

func (f *Frontend) Terminate() {
	f.Conn.Close()
	f = nil
}

func (f *Frontend) listen() {
	defer f.Terminate()
	conn := f.Conn
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			return
		}

		frontendaction.Execute(f, msg)
	}
}

func (f *Frontend) heartbeat() {
	ticker := time.NewTicker(pingPeriod)
	conn := f.Conn
	defer ticker.Stop()
	defer conn.Close()
	for {
		<-ticker.C
		conn.SetWriteDeadline(time.Now().Add(writeTimeout))
		if err := conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
			return
		}
	}
}
