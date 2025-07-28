package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
)

type Request struct {
	ID      int                 `json:"id"`
	IP      string              `json:"ip"`
	Method  string              `json:"method"`
	URL     string              `json:"url"`
	Headers map[string][]string `json:"headers"`
	Body    []byte              `json:"body"`
}

type Response struct {
	ID         int                 `json:"id"`
	StatusCode int                 `json:"status_code"`
	Headers    map[string][]string `json:"headers"`
	Body       []byte              `json:"body"`
	Err        string              `json:"err"`
}

var lastId = 0

func makeRequest(req *http.Request) (*Request, error) {
	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	realIp := req.Header.Get("X-Forwarded-For")
	if realIp == "" {
		realIp = req.Header.Get("X-Real-Ip")
	}
	if realIp == "" {
		realIp = req.RemoteAddr
	}
	lastId++
	return &Request{
		ID:      lastId,
		IP:      realIp,
		Method:  req.Method,
		URL:     req.URL.String(),
		Headers: req.Header,
		Body:    bodyBytes,
	}, nil
}

type responses struct {
	mp map[int]chan *Response
	mu sync.Mutex
}

func MakeProxyResponses() ProxyResponses {
	return &responses{
		mp: make(map[int]chan *Response),
	}
}

type ProxyResponses interface {
	Add(*http.Request) (chan *Response, *Request, error)
	Remove(id int)
	Response(data []byte) error
	CloseAll()
}

func (rs *responses) Add(r *http.Request) (chan *Response, *Request, error) {
	req, err := makeRequest(r)
	if err != nil {
		return nil, nil, err
	}

	ch := make(chan *Response, 1)
	rs.mu.Lock()
	rs.mp[req.ID] = ch
	rs.mu.Unlock()

	return ch, req, nil
}

func (rs *responses) Remove(id int) {
	rs.mu.Lock()
	ch, ok := rs.mp[id]
	if ok {
		close(ch)
		delete(rs.mp, id)
	}
	rs.mu.Unlock()
}

func (rs *responses) CloseAll() {
	rs.mu.Lock()
	for id, v := range rs.mp {
		select {
		case v <- &Response{
			ID:  id,
			Err: "connection was closed",
		}:
		default:
		}
	}
	rs.mu.Unlock()
}

func (rs *responses) get(id int) (ch chan *Response, ok bool) {
	rs.mu.Lock()
	ch, ok = rs.mp[id]
	rs.mu.Unlock()
	return
}

func (rs *responses) Response(data []byte) error {
	var res Response
	if err := json.Unmarshal(data, &res); err != nil {
		return err
	}
	ch, ok := rs.get(res.ID)
	if !ok {
		return fmt.Errorf("Response handler not found")
	}
	ch <- &res
	return nil
}

func (r *Request) ToHttp() (*http.Request, error) {
	req, err := http.NewRequest(r.Method, r.URL, bytes.NewReader(r.Body))
	if err != nil {
		return nil, err
	}

	for k, v := range r.Headers {
		for _, h := range v {
			req.Header.Add(k, h)
		}
	}

	req.Header.Set("X-Forwarded-For", r.IP)
	req.Header.Set("X-Real-Ip", r.IP)

	return req, nil
}

func (r *Request) ResponseError(err error) ([]byte, error) {
	respData := &Response{
		ID:  r.ID,
		Err: err.Error(),
	}
	return json.Marshal(respData)
}

func (r *Request) Response(res *http.Response) ([]byte, error) {
	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	respData := &Response{
		ID:         r.ID,
		StatusCode: res.StatusCode,
		Headers:    res.Header,
		Body:       bodyBytes,
	}
	return json.Marshal(respData)
}

func ParseRequest(data []byte) (*Request, error) {
	var req Request
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, err
	}
	return &req, nil
}
