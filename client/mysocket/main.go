package mysocket

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/websocket"
)

var loginfo = log.New(os.Stdout, "", log.Ltime)

type Websocket struct {
	url       string
	secret    string
	c         *websocket.Conn
	done      chan struct{}
	closed    bool
	OnTextMsg func([]byte)
}

func New(url, secret string) *Websocket {
	done := make(chan struct{}, 1)
	return &Websocket{url, secret, nil, done, false, func(b []byte) {}}
}

func (ws *Websocket) Close() {
	if ws.c != nil {
		ws.c.Close()
	}
}
func (ws *Websocket) CloseMessage() {
	if ws.c == nil {
		return
	}
	ws.closed = true
	err := ws.c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		log.Println("ws close:", err)
		ws.closeDone()
	}
}

func (ws *Websocket) WriteMessage(data []byte) {
	if ws.c == nil {
		return
	}
	err := ws.c.WriteMessage(websocket.TextMessage, data)
	if err != nil {
		log.Println("ws write:", err)
		return
	}
}

func (ws *Websocket) Done() chan struct{} {
	return ws.done
}

func (ws *Websocket) Run() {
	defer ws.closeDone()
	ws.dial()
	if ws.c == nil {
		return
	}
	loginfo.Println("waiting for requests...")
	for {
		mt, message, err := ws.c.ReadMessage()
		if err != nil {
			log.Println("ws read:", err)
			ws.reconnect()
			if ws.c == nil {
				return
			}
			continue
		}
		if mt != websocket.TextMessage {
			continue
		}
		go ws.OnTextMsg(message)
	}
}

func (ws *Websocket) dial() {
	wsc, _, err := websocket.DefaultDialer.Dial(ws.url, http.Header{
		"Authorization": []string{ws.secret},
	})
	ws.c = wsc
	if err != nil {
		log.Println("ws dial:", err)
	}
}

func (ws *Websocket) closeDone() {
	select {
	case ws.done <- struct{}{}:
	default:
	}
}

func (ws *Websocket) reconnect() {
	ws.c.Close()
	ws.c = nil
	if ws.closed {
		return
	}
	loginfo.Println("ws reconnecting...")
	for _, i := range []time.Duration{1, 2, 5, 10, 20, 40, 60, 0} {
		t := time.After(time.Millisecond * (50 * i))
		ws.dial()
		if ws.c != nil {
			loginfo.Println("waiting for requests...")
			return
		}
		<-t
	}
	log.Println("cant reconnect")
}
