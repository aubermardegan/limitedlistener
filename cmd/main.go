package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"time"

	"github.com/aubermardegan/limitedlistener"
)

type Server struct {
	ln          *limitedlistener.LimitedListener
	bytesReadCh chan int
	quit        chan struct{}
}

func NewServer() *Server {
	return &Server{
		bytesReadCh: make(chan int),
		quit:        make(chan struct{}),
	}
}

func (s *Server) Start(global, perConn int) error {
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		return err
	}
	defer listener.Close()

	limitedlistener, err := limitedlistener.NewLimitedListener(listener, global, perConn)
	if err != nil {
		log.Fatal(err)
	}

	s.ln = limitedlistener
	fmt.Println("Listening on port 8080")

	go s.acceptLoop()

	<-s.quit
	close(s.bytesReadCh)
	return nil
}

func (s *Server) acceptLoop() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			fmt.Println("accept error:", err)
			continue
		}

		fmt.Println("New Connection")

		go s.readLoop(conn)
	}
}

func (s *Server) readLoop(conn net.Conn) {
	defer conn.Close()

	buf := make([]byte, 1_000_000)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			if err != io.EOF {
				fmt.Println("read error: ", err)
				continue
			}
			break
		}

		s.bytesReadCh <- n
	}
}

func sendDataToServer(fileSize int) {
	//Connect to the server.
	conn, err := net.Dial("tcp", ":8080")
	if err != nil {
		log.Fatalf("Failed to dial: %v", err)
	}
	defer conn.Close()

	//Send the data
	data := make([]byte, fileSize)
	_, err = conn.Write(data)
	if err != nil {
		fmt.Printf("write failed: %v", err)
	}
}

func logProgress(elapsed float64, totalBytesRead, targetRate int) {
	actualRate := float64(totalBytesRead) / elapsed

	lowerBound := float64(targetRate) * (0.95)
	upperBound := float64(targetRate) * (1.05)

	if actualRate < lowerBound || actualRate > upperBound {
		fmt.Printf("\nActual rate (%.2f bytes/s) is not within 5%% of target rate (%d bytes/s)", actualRate, targetRate)
	} else {
		fmt.Printf("\nActual rate (%.2f bytes/s) is within 5%% of target rate (%d bytes/s)", actualRate, targetRate)
	}
}

func samplingBandWidth() {
	targetRate := 1024          // 1 KB/s
	fileSize := 1 * 1024 * 1024 // 10 MB

	server := NewServer()
	go func() {
		log.Fatal(server.Start(targetRate, targetRate))
	}()
	time.Sleep(1 * time.Second)

	startTime := time.Now()
	sendDataToServer(fileSize)

	ticker := *time.NewTicker(30 * time.Second)

	transferFinished := make(chan struct{})
	totalBytesRead := 0

	for {
		select {
		case bytesRead := <-server.bytesReadCh:
			totalBytesRead += bytesRead
			if totalBytesRead == fileSize {
				go func() { transferFinished <- struct{}{} }()
			}
			fmt.Printf("\n%s -> Read %d bytes", time.Now().Format(time.DateTime), bytesRead)

		case <-ticker.C:
			logProgress(time.Since(startTime).Seconds(), totalBytesRead, targetRate)

		case <-transferFinished:
			fmt.Println("Finished!")
			logProgress(time.Since(startTime).Seconds(), totalBytesRead, targetRate)
			return
		}
	}
}

func changingLimitsOnRunTime() {
	targetRate := 1024          // 1 KB/s
	fileSize := 1 * 1024 * 1024 // 10 MB

	server := NewServer()
	go func() {
		log.Fatal(server.Start(targetRate, targetRate))
	}()

	time.Sleep(1 * time.Second)

	sendDataToServer(fileSize)

	tickerIncrease := time.NewTicker(10 * time.Second)
	tickerReduce := time.NewTicker(15 * time.Second)

	transferFinished := make(chan struct{})
	totalBytesRead := 0

	for {
		select {
		case bytesRead := <-server.bytesReadCh:
			totalBytesRead += bytesRead
			if totalBytesRead == fileSize {
				go func() { transferFinished <- struct{}{} }()
			}
			fmt.Printf("\n%s -> Read %d bytes", time.Now().Format(time.DateTime), bytesRead)

		case <-tickerIncrease.C:
			targetRate = targetRate * 2
			server.ln.SetLimits(targetRate, targetRate)
			fmt.Printf("\n%s -> Doubling the Limit", time.Now().Format(time.DateTime))

		case <-tickerReduce.C:
			targetRate = targetRate / 2
			server.ln.SetLimits(targetRate, targetRate)
			fmt.Printf("\n%s -> Halving the Limit", time.Now().Format(time.DateTime))

		case <-transferFinished:
			fmt.Printf("\n%s -> Finished!", time.Now().Format(time.DateTime))
			return
		}
	}
}

func main() {
	samplingBandWidth()
	//changingLimitsOnRunTime()
}
