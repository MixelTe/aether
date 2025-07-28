package mysocket

import (
	"log"
	"os"

	"github.com/gorilla/websocket"
)

var loginfo = log.New(os.Stdout, "", log.Ltime)

type Websocket struct {
	url       string
	c         *websocket.Conn
	done      chan struct{}
	OnTextMsg func([]byte)
}

func New(url string) *Websocket {
	done := make(chan struct{}, 1)
	return &Websocket{url, nil, done, func(b []byte) {}}
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
			return
		}
		if mt != websocket.TextMessage {
			continue
		}
		go ws.OnTextMsg(message)
	}
}

func (ws *Websocket) dial() {
	wsc, _, err := websocket.DefaultDialer.Dial(ws.url, nil)
	ws.c = wsc
	if err != nil {
		log.Fatal("ws dial:", err)
	}
}

func (ws *Websocket) closeDone() {
	select {
	case ws.done <- struct{}{}:
	default:
	}
}
