package scanner_test

import (
	"errors"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/idiomat/dodtnyt/e2/scanner"
)

func TestNewTCPScanner(t *testing.T) {
	tests := map[string]struct {
		workers int
		dialer  scanner.Dialer
		wantErr bool
	}{
		"valid configuration": {
			workers: 2,
			dialer:  &MockDialer{},
			wantErr: false,
		},
		"invalid number of workers": {
			workers: 0,
			dialer:  &MockDialer{},
			wantErr: true,
		},
		"nil dialer": {
			workers: 2,
			dialer:  nil,
			wantErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := scanner.NewTCPScanner("localhost", tt.workers, tt.dialer)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewTCPScanner(%d, %v) error = %v, wantErr %v", tt.workers, tt.dialer, err, tt.wantErr)
			}
		})
	}
}

func TestTCPScanner_Scan(t *testing.T) {
	tests := map[string]struct {
		openPorts         map[int]bool
		portsToScan       []int
		expectedOpenPorts []int
	}{
		"mixed open and closed ports": {
			openPorts:         map[int]bool{80: true, 81: false, 82: true},
			portsToScan:       []int{80, 81, 82},
			expectedOpenPorts: []int{80, 82},
		},
		"all ports closed": {
			openPorts:         map[int]bool{80: false, 81: false, 82: false},
			portsToScan:       []int{80, 81, 82},
			expectedOpenPorts: []int{},
		},
		"all ports open": {
			openPorts:         map[int]bool{80: true, 81: true, 82: true},
			portsToScan:       []int{80, 81, 82},
			expectedOpenPorts: []int{80, 81, 82},
		},
		"no ports to scan": {
			openPorts:         map[int]bool{80: true, 81: false, 82: true},
			portsToScan:       []int{},
			expectedOpenPorts: []int{},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mockDialer := &MockDialer{
				openPorts: tt.openPorts,
			}
			scanner, err := scanner.NewTCPScanner("127.0.0.1", scanner.DefaultNumWorkers, mockDialer)
			if err != nil {
				t.Fatalf("failed to create scanner: %v", err)
			}

			openPorts, err := scanner.Scan(tt.portsToScan)
			if err != nil {
				t.Errorf("TCPScanner.Scan() error = %v", err)
			}

			if !equal(openPorts, tt.expectedOpenPorts) {
				t.Errorf("TCPScanner.Scan() = %v, want %v", openPorts, tt.expectedOpenPorts)
			}
		})
	}
}

func equal(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}

	// Create a map to count occurrences of each element in 'a'
	counts := make(map[int]int)
	for _, v := range a {
		counts[v]++
	}

	// Check elements in 'b' against the map
	for _, v := range b {
		if counts[v] == 0 {
			return false
		}
		counts[v]--
	}

	return true
}

// MockConn is a mock implementation of the net.Conn interface.
type MockConn struct{}

func (mc *MockConn) Read(b []byte) (n int, err error) {
	return 0, nil
}

func (mc *MockConn) Write(b []byte) (n int, err error) {
	return len(b), nil
}

func (mc *MockConn) Close() error {
	return nil
}

func (mc *MockConn) LocalAddr() net.Addr {
	return &net.IPAddr{IP: net.IPv4(127, 0, 0, 1)}
}

func (mc *MockConn) RemoteAddr() net.Addr {
	return &net.IPAddr{IP: net.IPv4(127, 0, 0, 1)}
}

func (mc *MockConn) SetDeadline(t time.Time) error {
	return nil
}

func (mc *MockConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (mc *MockConn) SetWriteDeadline(t time.Time) error {
	return nil
}

// MockDialer is a mock implementation of the dialer interface.
type MockDialer struct {
	openPorts map[int]bool
}

func (m *MockDialer) Dial(network, address string) (net.Conn, error) {
	var port int
	fmt.Sscanf(address, "127.0.0.1:%d", &port)
	if m.openPorts[port] {
		return &MockConn{}, nil
	}
	return nil, errors.New("connection refused")
}
