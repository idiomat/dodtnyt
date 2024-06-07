package scanner

import (
	"fmt"
	"net"
	"runtime"
	"strings"
	"sync"
	"time"
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

func (s *TCPScanner) Scan(ports []int) ([]int, error) {
	// The done channel will be shared by the entire pipeline
	// so that when it's closed it serves as a signal
	// for all the goroutines we started to exit.
	done := make(chan struct{})
	defer close(done)

	in := s.gen(done, ports...)

	// fan-out
	var chans []<-chan scanOp
	for i := 0; i < s.workers; i++ {
		chans = append(chans, s.scan(done, in))
	}

	var openPorts []int

	for s := range s.filterOpen(done, s.merge(done, chans...)) {
		openPorts = append(openPorts, s.port)
	}

	// for s := range s.filterErr(done, s.merge(done, chans...)) {
	// 	fmt.Printf("%#v\n", s)
	// 	done <- struct{}{}
	// }

	// done chan is closed by the deferred call here

	return openPorts, nil
}

type scanOp struct {
	port         int
	open         bool
	scanErr      string
	scanDuration time.Duration
}

func (s *TCPScanner) gen(done <-chan struct{}, ports ...int) <-chan scanOp {
	out := make(chan scanOp, len(ports))
	go func() {
		defer close(out)
		for _, p := range ports {
			select {
			case out <- scanOp{port: p}:
			case <-done:
				return
			}
		}
	}()
	return out
}

func (s *TCPScanner) scan(done <-chan struct{}, in <-chan scanOp) <-chan scanOp {
	out := make(chan scanOp)
	go func() {
		defer close(out)
		for scan := range in {
			select {
			default:
				address := fmt.Sprintf("%s:%d", s.host, scan.port)
				start := time.Now()
				conn, err := s.dialer.Dial("tcp", address)
				scan.scanDuration = time.Since(start)
				if err != nil {
					scan.scanErr = err.Error()
				} else {
					conn.Close()
					scan.open = true
				}
				out <- scan
			case <-done:
				return
			}
		}
	}()
	return out
}

func (s *TCPScanner) filterOpen(done <-chan struct{}, in <-chan scanOp) <-chan scanOp {
	out := make(chan scanOp)
	go func() {
		defer close(out)
		for scan := range in {
			select {
			default:
				if scan.open {
					out <- scan
				}
			case <-done:
				return
			}
		}
	}()
	return out
}

func (s *TCPScanner) filterErr(done <-chan struct{}, in <-chan scanOp) <-chan scanOp {
	out := make(chan scanOp)
	go func() {
		defer close(out)
		for scan := range in {
			select {
			default:
				if !scan.open && strings.Contains(scan.scanErr, "too many open files") {
					out <- scan
				}
			case <-done:
				return
			}
		}
	}()
	return out
}

func (s *TCPScanner) merge(done <-chan struct{}, chans ...<-chan scanOp) <-chan scanOp {
	out := make(chan scanOp)
	wg := sync.WaitGroup{}
	wg.Add(len(chans))

	for _, sc := range chans {
		go func(sc <-chan scanOp) {
			defer wg.Done()
			for scan := range sc {
				select {
				case out <- scan:
				case <-done:
					return
				}
			}
		}(sc)
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}
