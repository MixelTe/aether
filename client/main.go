package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"time"

	"aether/client/mysocket"
	"aether/server/proxy"
)

var (
	port     = flag.String("port", "", "local forwarding port (required)")
	ws       *mysocket.Websocket
	localUrl *url.URL
	loginfo  = log.New(os.Stdout, "", log.Ltime)
)

func main() {
	flag.Parse()
	log.SetFlags(log.Ltime)
	cfg := loadConfig()
	if *port == "" {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
		return
	}

	var err error
	localUrl, err = url.Parse("http://127.0.0.1:" + *port)
	if err != nil {
		log.Printf("error in parse addr: %v", err)
		return
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	Scheme := "ws"
	if cfg.Usewss {
		Scheme = "wss"
	}
	u := url.URL{Scheme: Scheme, Host: cfg.Host, Path: "/aether/client/ws"}
	loginfo.Printf("forwarding to :%v", *port)
	loginfo.Printf("connecting to %s", u.String())

	ws = mysocket.New(u.String(), cfg.Secret)
	ws.OnTextMsg = processRequest
	defer ws.Close()

	go ws.Run()

	for {
		select {
		case <-ws.Done():
			return
		case <-interrupt:
			loginfo.Println("interrupt")
			ws.CloseMessage()
			select {
			case <-ws.Done():
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

	res, code, err := sendRequest(req)
	if err != nil {
		log.Println("req: ", err)
		jsonBytes, err := req.ResponseError(err)
		if err == nil {
			ws.WriteMessage(jsonBytes)
		}
		return
	}

	loginfo.Printf("[%v] %v %v", req.Method, code, req.URL)
	ws.WriteMessage(res)
}

func sendRequest(r *proxy.Request) ([]byte, int, error) {
	req, err := r.ToHttp()
	if err != nil {
		return nil, 0, err
	}

	req.URL.Scheme = localUrl.Scheme
	req.URL.Host = localUrl.Host

	res, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		return nil, 0, err
	}

	resp, err := r.Response(res)
	return resp, res.StatusCode, err
}
