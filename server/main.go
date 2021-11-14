package main

import (
	"bufio"
	"log"
	"net"
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

		for item := range openConnections {
			if item != conn {
				item.Write([]byte(message))
			}
		}
	}

	deadConnection <- conn
}
