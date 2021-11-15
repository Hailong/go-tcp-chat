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

func main() {
	go func() {
		if err := http.ListenAndServe("localhost:6060", nil); err != nil {
			log.Fatalf("Pprof failed: %v", err)
		}
	}()

	var (
		// == SERVER ==
		PORT = 8000
		ADDR = [4]byte{127, 0, 0, 1}
		// ============
		LISTENBACKLOG = 100
		MAXMSGSIZE    = 8000
	)

	serverFD, err := unix.Socket(unix.AF_INET, unix.SOCK_STREAM, unix.IPPROTO_IP)
	logFatal(err)

	serverAddr := &unix.SockaddrInet4{
		Port: PORT,
		Addr: ADDR,
	}

	err = unix.Bind(serverFD, serverAddr)
	logFatal(err)

	fmt.Printf("Server: Bound to addr: %d, port: %d\n", serverAddr.Addr, serverAddr.Port)

	err = unix.Listen(serverFD, LISTENBACKLOG)
	logFatal(err)

	var activeFdSet unix.FdSet
	var tmpFdSet unix.FdSet
	var fdMax int
	FDZero(&activeFdSet)
	FDSet(serverFD, &activeFdSet)
	fdMax = serverFD

	fdAddr := FDAddrInit()

	for {
		tmpFdSet = activeFdSet

		_, err := unix.Select(fdMax+1, &tmpFdSet, nil, nil, nil)
		logFatal(err)

		for fd := 0; fd < fdMax+1; fd++ {
			if FDIsSet(fd, &tmpFdSet) {
				if fd == serverFD {
					acceptedFD, acceptedAddr, err := unix.Accept(serverFD)
					logFatal(err)

					FDSet(acceptedFD, &activeFdSet)
					fdAddr.Set(acceptedFD, acceptedAddr)
					if acceptedFD > fdMax {
						fdMax = acceptedFD
					}
				} else {
					msg := make([]byte, MAXMSGSIZE)

					sizeMsg, _, err := unix.Recvfrom(fd, msg, 0)

					if err != nil {
						fmt.Println("Recvfrom: ", err)
						FDClr(fd, &activeFdSet)
						unix.Close(fd)
						fdAddr.Clr(fd)
						continue
					}

					clientAddr := fdAddr.Get(fd)
					addrFrom := clientAddr.(*unix.SockaddrInet4)
					fmt.Printf("%d byte read from %d:%d on socket %d\n",
						sizeMsg, addrFrom.Addr, addrFrom.Port, fd)
					print("> Received message:\n" + string(msg) + "\n")
					response := []byte("We just received your message: " + string(msg))

					err = unix.Sendmsg(
						fd,
						response,
						nil, clientAddr, unix.MSG_DONTWAIT,
					)
					logFatal(err)

					print("< Response message:\n" + string(response) + "\n")

					FDClr(fd, &activeFdSet)
					fdAddr.Clr(fd)
					unix.Close(fd)
				}
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
