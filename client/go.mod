module aether/client

go 1.24.5

replace aether/server => ../server

require (
	aether/server v0.0.0-00010101000000-000000000000
	github.com/gorilla/websocket v1.5.3
)
