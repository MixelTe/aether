package main

import (
	"flag"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"time"

	"aether/server/proxy"

	"github.com/gorilla/websocket"
)

var (
	port     = flag.String("port", "8080", "local forwarding port")
	addr     = flag.String("addr", "localhost:8000", "http service address")
	wsc      *websocket.Conn
	localUrl *url.URL
)

func main() {
	flag.Parse()

	var err error
	localUrl, err = url.Parse("http://127.0.0.1:" + *port)
	if err != nil {
		log.Printf("error in parse addr: %v", err)
		return
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	u := url.URL{Scheme: "ws", Host: *addr, Path: "/aether/client/ws"}
	log.Printf("connecting to %s", u.String())

	wsc, _, err = websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer wsc.Close()

	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			mt, message, err := wsc.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}
			if mt != websocket.TextMessage {
				continue
			}
			go processRequest(message)
		}
	}()

	for {
		select {
		case <-done:
			return
		case <-interrupt:
			log.Println("interrupt")

			err := wsc.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("write close:", err)
				return
			}
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return
		}
	}
}

func processRequest(message []byte) {
	req, err := proxy.ParseRequest(message)
	if err != nil {
		log.Println("req: ", err)
		return
	}

	res, err := sendRequest(req)
	if err != nil {
		log.Println("req: ", err)
		jsonBytes, err := req.ResponseError(err)
		if err == nil {
			wsc.WriteMessage(websocket.TextMessage, jsonBytes)
		}
		return
	}

	wsc.WriteMessage(websocket.TextMessage, res)
}

func sendRequest(r *proxy.Request) ([]byte, error) {
	req, err := r.ToHttp()
	if err != nil {
		return nil, err
	}

	req.URL.Scheme = localUrl.Scheme
	req.URL.Host = localUrl.Host

	res, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	return r.Response(res)
}
