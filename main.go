package main

import (
	"fmt"
	"log"
	"net"

	"github.com/aubermardegan/limitedlistener/limitedlistener"
)

func main() {

	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal(err)
	}

	limitedlistener, err := limitedlistener.NewLimitedListener(listener, 1000, 1000)
	if err != nil {
		log.Fatal(err)
	}

	for {
		conn, err := limitedlistener.Accept()
		if err != nil {
			fmt.Println(err)
			continue
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	buf := make([]byte, 1024)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			break
		}
		_, err = conn.Write(buf[:n])
		if err != nil {
			fmt.Printf("write error: %v\n", err)
		}
	}
}
