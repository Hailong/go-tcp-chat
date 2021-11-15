package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
)

func logFatal(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

var (
	openConnections = make(map[net.Conn]bool)
	newConnection   = make(chan net.Conn)
	deadConnection  = make(chan net.Conn)
)

func main() {
	ln, err := net.Listen("tcp", ":8000")
	logFatal(err)

	defer ln.Close()

	go func() {
		for {
			conn, err := ln.Accept()
			logFatal(err)

			openConnections[conn] = true
			newConnection <- conn
		}
	}()

	for {
		select {
		case conn := <-newConnection:
			go broadcastMessage(conn)
		case conn := <-deadConnection:
			for item := range openConnections {
				if item == conn {
					break
				}
			}

			delete(openConnections, conn)
		}
	}
}

func broadcastMessage(conn net.Conn) {
	for {
		reader := bufio.NewReader(conn)
		message, err := reader.ReadString('\n')

		if err != nil {
			break
		}

		if strings.HasPrefix(message, "GET") {
			response := handleHttpRequest(message)

			conn.Write(response)

			break
		} else {
			for item := range openConnections {
				if item != conn {
					item.Write([]byte(message))
				} else {
					item.Write([]byte(fmt.Sprint("Received: ", message)))
				}
			}
		}
	}

	deadConnection <- conn
}

var httpMessageTemplate = `HTTP/1.1 %d OK
Cache-Control: no-cache, private
Content-Type: %s
Content-Length: %d

`

func handleHttpRequest(message string) []byte {
	statusCode := 200
	contentType := "text/html"

	var (
		content []byte
		err     error
	)

	switch {
	case strings.HasPrefix(message, "GET / HTTP"):
		content, err = os.ReadFile("./wwwroot/index.html")
		logFatal(err)
	case strings.HasPrefix(message, "GET /favicon.ico HTTP"):
		contentType = "image/vnd.microsoft.icon"
		content, err = os.ReadFile("./wwwroot/favicon.ico")
		logFatal(err)
	default:
		statusCode = 404
		content, err = os.ReadFile("./wwwroot/404.html")
		logFatal(err)
	}

	header := []byte(fmt.Sprintf(
		httpMessageTemplate,
		statusCode,
		contentType,
		len(content),
	))

	return append(header, content...)
}
