package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strings"

	"github.com/gobwas/ws/wsutil"
	"golang.org/x/sys/unix"
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

var epoller *epoll

func main() {
	go func() {
		if err := http.ListenAndServe("localhost:6060", nil); err != nil {
			log.Fatalf("Pprof failed: %v", err)
		}
	}()

	// start epoll
	var err error
	epoller, err = MkEpoll()
	logFatal(err)

	go Start()

	ln, err := net.Listen("tcp", ":8000")
	logFatal(err)

	for {
		conn, err := ln.Accept()
		logFatal(err)

		if err := epoller.Add(conn); err != nil {
			log.Printf("Failed to add connection")
			conn.Close()
		}
	}
}

func Start() {
	for {
		connections, err := epoller.Wait()
		if err != nil {
			log.Printf("Failed to epoll wait %v", err)
			continue
		}
		for _, conn := range connections {
			if conn == nil {
				break
			}
			if msg, _, err := wsutil.ReadClientData(conn); err != nil {
				if err := epoller.Remove(conn); err != nil {
					log.Printf("Failed to remove %v", err)
				}
				conn.Close()
			} else {
				// This is commented out since in demo usage, stdout is showing messages sent from > 1M connections at very high rate
				log.Printf("msg: %s", string(msg))
			}
		}
	}
}

// FdSet store the active FDs
// type unix.FdSet struct {
//     Bits [32]int32 // FD_SETSIZE = 1024 = 32x32
// }

// FDZero set to zero the fdSet
func FDZero(p *unix.FdSet) {
	p.Bits = [32]int32{}
}

// FDSet set a fd of fdSet
func FDSet(fd int, p *unix.FdSet) {
	p.Bits[fd/32] |= (1 << (uint(fd) % 32))
}

// FDClr clear a fd of fdSet
func FDClr(fd int, p *unix.FdSet) {
	p.Bits[fd/32] &^= (1 << (uint(fd) % 32))
}

// FDIsSet return true if fd is set
func FDIsSet(fd int, p *unix.FdSet) bool {
	return p.Bits[fd/32]&(1<<(uint(fd)%32)) != 0
}

// FDAddr is the type storing the sockaddr of each fd
type FDAddr map[int]unix.Sockaddr

// FDAddrInit init FDAddr with the size of FDSize
func FDAddrInit() *FDAddr {
	f := make(FDAddr, unix.FD_SETSIZE)
	return &f
}

// Get return the Sockaddr value of a given fd key
func (f *FDAddr) Get(fd int) unix.Sockaddr {
	return (*f)[fd]
}

// Set set the Sockaddr value of a given fd key
func (f *FDAddr) Set(fd int, value unix.Sockaddr) {
	(*f)[fd] = value
}

// Clr remove a given fd key in FDAddr
func (f *FDAddr) Clr(fd int) {
	delete(*f, fd)
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
