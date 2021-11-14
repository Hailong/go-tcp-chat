# go-tcp-chat

Exploring TCP programming in Go with goroutine, kqueue, epoll, and their performance difference.

### Iteration 1

A basic skeleton of TCP chat client/server that based on goroutine

### Iteration 2

Added the support of serving HTTP request

### Iteration 3

Added [pprof](https://github.com/google/pprof) code to test the performance

### Iteration 4

Optimize the server with kqueue

### Iteration 5

Optimize the server with epoll

_NOTE: iteration 4 and 5 has some glitches so far, still working on it_

### Resources

- [Going Infinite, handling 1 millions websockets connections in Go / Eran Yanay](https://youtu.be/LI1YTFMi8W4)
- [Section 4: Golang TCP chat: establish client server connection](https://youtu.be/waZt518cxFI)
- [Building a Web Server in C++ [VS 2017] Part 2](https://youtu.be/YqEqjODUkWY)
- [Minimalist TCP server/client in Go using only syscalls - Select() to handle file descriptors](https://gist.github.com/vomnes/be42868583db5812b7266b2f45262dca)
- [https://github.com/cppis/elio](https://github.com/cppis/elio)
