package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/url"
	"os"
	"time"
)

var (
	ip          = flag.String("ip", "127.0.0.1", "server IP")
	connections = flag.Int("conn", 1, "number of tcp connections")
)

func main() {
	flag.Usage = func() {
		io.WriteString(os.Stderr, `Go TCP chat server load test client

Example usage: ./client -ip=172.17.0.1 -conn=10
`)
		flag.PrintDefaults()
	}
	flag.Parse()

	u := url.URL{Scheme: "tcp", Host: *ip + ":8000"}
	log.Printf("Connecting to %s", u.String())

	var conns []net.Conn

	for i := 0; i < *connections; i++ {
		c, err := net.Dial(u.Scheme, u.Host)
		if err != nil {
			fmt.Println("Failed to connect", i, err)
			break
		}

		conns = append(conns, c)

		defer func() {
			time.Sleep(time.Second)
			c.Close()
		}()
	}

	log.Printf("Finished initializing %d connections", len(conns))

	tts := time.Second
	if *connections > 100 {
		tts = time.Millisecond * 5
	}

	for {
		for i := 0; i < len(conns); i++ {
			time.Sleep(tts)
			conn := conns[i]

			log.Printf("Conn %d sending message", i)

			conn.Write([]byte(fmt.Sprintf("Hello from conn %v", i)))
		}
	}
}
