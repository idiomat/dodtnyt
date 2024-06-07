package main

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/idiomat/dodtnyt/testing/concurrent/scanner"
)

var host string
var ports string
var numWorkers int

func init() {
	flag.StringVar(&host, "host", "127.0.0.1", "Host to scan.")
	flag.StringVar(&ports, "ports", "80", "Port(s) (e.g. 80, 22-100).")
	flag.IntVar(&numWorkers, "workers", runtime.NumCPU(), "Number of workers. Defaults to 10.")
}

func main() {
	flag.Parse()

	portsToScan, err := parsePortsToScan(ports)
	if err != nil {
		fmt.Printf("failed to parse ports to scan: %s\n", err)
		os.Exit(1)
	}

	tcpScanner, err := scanner.NewTCPScanner(host, numWorkers, &net.Dialer{})
	if err != nil {
		fmt.Printf("failed to create TCP scanner: %s\n", err)
		os.Exit(1)
	}

	openPorts, err := tcpScanner.Scan(portsToScan)
	if err != nil {
		fmt.Printf("failed to scan ports: %s\n", err)
		os.Exit(1)
	}

	fmt.Println("RESULTS")
	sort.Ints(openPorts)
	for _, p := range openPorts {
		fmt.Printf("%d - open\n", p)
	}
}

// parsePortsToScan parses the ports string and returns a slice of ports to scan.
func parsePortsToScan(portsFlag string) ([]int, error) {
	p, err := strconv.Atoi(portsFlag)
	if err == nil {
		return []int{p}, nil
	}

	ports := strings.Split(portsFlag, "-")
	if len(ports) != 2 {
		return nil, errors.New("unable to determine port(s) to scan")
	}

	minPort, err := strconv.Atoi(ports[0])
	if err != nil {
		return nil, fmt.Errorf("failed to convert %s to a valid port number", ports[0])
	}

	maxPort, err := strconv.Atoi(ports[1])
	if err != nil {
		return nil, fmt.Errorf("failed to convert %s to a valid port number", ports[1])
	}

	if minPort <= 0 || maxPort <= 0 {
		return nil, fmt.Errorf("port numbers must be greater than 0")
	}

	var results []int
	for p := minPort; p <= maxPort; p++ {
		results = append(results, p)
	}
	return results, nil
}
