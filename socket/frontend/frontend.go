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
	f.Conn.SetWriteDeadline(time.Now().Add(writeTimeout))
	if err := f.Conn.WriteMessage(websocket.TextMessage, data); err != nil {
		f.Terminate()
	}
}

func (f *Frontend) Terminate() {
	f.Conn.Close()
}

func (f *Frontend) listen() {
	defer f.Terminate()
	for {
		_, msg, err := f.Conn.ReadMessage()
		if err != nil {
			return
		}

		// Call appropraite action from package hybris/frontend/frontendaction
		frontendaction.Execute(f, msg)
	}
}

func (f *Frontend) heartbeat() {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	defer f.Conn.Close()
	for {
		<-ticker.C
		f.Conn.SetWriteDeadline(time.Now().Add(writeTimeout))
		if err := f.Conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
			return
		}
	}
}
