package scanner

import (
	"fmt"
	"net"
	"runtime"
)

type Dialer interface {
	Dial(network, address string) (net.Conn, error)
}

var DefaultNumWorkers = runtime.NumCPU()

type TCPScanner struct {
	host    string
	workers int
	dialer  Dialer
}

func (s *TCPScanner) validate() error {
	if s.workers < 1 {
		return fmt.Errorf("invalid number of workers: %d", s.workers)
	}
	if s.dialer == nil {
		return fmt.Errorf("dialer is required")
	}
	return nil
}

func NewTCPScanner(host string, workers int, dialer Dialer) (*TCPScanner, error) {
	s := &TCPScanner{host: host, workers: workers, dialer: dialer}
	return s, s.validate()
}

type scanner interface {
	Scan(ports []int) ([]int, error)
}

// Compile-time check to verify TCPScanner implements the Scanner interface.
var _ scanner = &TCPScanner{}

// Scan scans the specified ports.
func (s *TCPScanner) Scan(ports []int) ([]int, error) {
	portsChan := make(chan int, s.workers)
	resultsChan := make(chan int)

	for i := 0; i < s.workers; i++ {
		go s.worker(portsChan, resultsChan)
	}

	go func() {
		for _, p := range ports {
			portsChan <- p
		}
		close(portsChan)
	}()

	var openPorts []int
	for i := 0; i < len(ports); i++ {
		if p := <-resultsChan; p != 0 {
			openPorts = append(openPorts, p)
		}
	}
	close(resultsChan)

	return openPorts, nil
}

// worker scans ports and sends results to the results channel.
func (s *TCPScanner) worker(portsChan <-chan int, resultsChan chan<- int) {
	for p := range portsChan {
		if s.scan(p) {
			resultsChan <- p
		} else {
			resultsChan <- 0
		}
	}
}

func (s *TCPScanner) scan(port int) bool {
	address := fmt.Sprintf("%s:%d", s.host, port)
	conn, err := s.dialer.Dial("tcp", address)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}
