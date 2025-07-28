package main

import (
	"aether/server/proxy"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var (
	addr      = flag.String("addr", ":8000", "http service address")
	upgrader  = websocket.Upgrader{} // use default option
	wsc       *websocket.Conn
	responses = proxy.MakeProxyResponses()
)

func main() {
	flag.Parse()
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	r := gin.Default()
	r.GET("/aether/client/ws", ws)
	r.NoRoute(prox)
	log.Fatal(r.Run(*addr))
}

func ws(c *gin.Context) {
	if wsc != nil {
		c.String(http.StatusServiceUnavailable, "Only one client at once")
		return
	}
	w, r := c.Writer, c.Request
	var err error
	wsc, err = upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("ws upgrade:", err)
		wsc = nil
		return
	}
	defer func() {
		responses.CloseAll()
		wsc.Close()
		wsc = nil
	}()
	for {
		mt, message, err := wsc.ReadMessage()
		if err != nil {
			log.Println("ws read:", err)
			break
		}
		if mt != websocket.TextMessage {
			continue
		}
		err = responses.Response(message)
		if err != nil {
			log.Println("ws resp:", err)
		}
	}
}

func prox(c *gin.Context) {
	if wsc == nil {
		c.AbortWithStatus(http.StatusServiceUnavailable)
		return
	}

	ch, req, err := responses.Add(c.Request)
	defer responses.Remove(req.ID)

	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		log.Println("proxy:", err)
		return
	}

	jsonBytes, err := json.Marshal(req)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		log.Println("proxy:", err)
		return
	}

	err = wsc.WriteMessage(websocket.TextMessage, jsonBytes)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		log.Println("proxy:", err)
		return
	}

	var res *proxy.Response
	select {
	case res = <-ch:
	case <-time.After(10 * time.Minute):
		c.AbortWithStatus(http.StatusRequestTimeout)
		return
	case <-c.Request.Context().Done():
		return
	}

	if res.Err != "" {
		c.AbortWithStatus(http.StatusInternalServerError)
		log.Println("proxy: client error:", res.Err)
		return
	}

	for k, vv := range res.Headers {
		for _, v := range vv {
			c.Header(k, v)
		}
	}
	c.Status(res.StatusCode)
	c.Writer.Write(res.Body)
}
