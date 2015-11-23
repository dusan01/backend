package client

import (
	"errors"
	"hybris/db/dbglobalban"
	"hybris/db/dbsession"
	"hybris/realtime"
	"hybris/socket/client/clientaction"
	"hybris/socket/message"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"gopkg.in/mgo.v2/bson"
	uppdb "upper.io/db"
)

const (
	// move to a new hybris/constants package?
	writeTimeout = 55 * time.Second
	pingPeriod   = 10 * time.Second
)

type Client struct {
	sync.Mutex
	Conn         *websocket.Conn
	ConnM        sync.Mutex
	RealtimeUser *realtime.User
	CommunityId  bson.ObjectId
}

var Clients = map[bson.ObjectId]*Client{}

func New(req *http.Request, conn *websocket.Conn) (*Client, error) {
	if realtime.Maintenance {
		conn.Close()
		return nil, errors.New("server is currently in maintenance mode")
	}

	cookie, err := req.Cookie("auth")
	if err != nil {
		conn.Close()
		return nil, errors.New("couldn't get auth cookie")
	}

	session, err := dbsession.Get(uppdb.Cond{"cookie": cookie.Value})
	if err != nil {
		conn.Close()
		return nil, errors.New("couldn't find session")
	}

	if globalBan, err := dbglobalban.Get(uppdb.Cond{"banneeId": session.UserId}); err == nil {
		if globalBan.Until == nil || globalBan.Until.After(time.Now()) {
			return nil, errors.New("banned")
		} else if err := globalBan.Delete(); err != nil {
			return nil, err
		}
	} else if err != uppdb.ErrNoMoreRows {
		return nil, err
	}

	c := &Client{
		Conn: conn,
	}

	if client, ok := Clients[session.UserId]; ok {
		client.Lock()
		defer client.Unlock()
		message.NewEvent("staleSession", true).Dispatch(client)
		c.RealtimeUser = client.RealtimeUser.Hijack(c)
	} else {
		c.RealtimeUser = realtime.NewUser(session.UserId, c)
	}

	Clients[session.UserId] = c

	message.NewUnique(message.S{
		"hello": true,
	}).Dispatch(c)

	go c.heartbeat()
	go c.listen()
	return c, nil
}

func (c *Client) Send(data []byte) {
	c.ConnM.Lock()
	defer c.ConnM.Unlock()
	c.Conn.SetWriteDeadline(time.Now().Add(writeTimeout))
	if err := c.Conn.WriteMessage(websocket.TextMessage, data); err != nil {
		c.Conn.Close()
	}
}

// Destroy client
// After 30 seconds;
//  Check if the client exists
//  If not then destroy the realtimeUser

func (c *Client) Terminate() {
	realtimeUser := c.RealtimeUser
	c.Conn.Close()
	delete(Clients, realtimeUser.Id)
	c = nil
	time.AfterFunc(30*time.Second, func() {
		if _, ok := Clients[realtimeUser.Id]; !ok && realtimeUser != nil {
			realtimeUser.Destroy()
		}
	})
}

func (c *Client) GetRealtimeUser() *realtime.User {
	return c.RealtimeUser
}

func (c *Client) listen() {
	defer c.Terminate()
	for {
		_, msg, err := c.Conn.ReadMessage()
		if err != nil {
			return
		}

		clientaction.Execute(c, msg)
	}
}

func (c *Client) heartbeat() {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	defer c.Conn.Close()
	for {
		<-ticker.C
		c.Conn.SetWriteDeadline(time.Now().Add(writeTimeout))
		if err := c.Conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
			return
		}
	}
}
