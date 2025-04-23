package telegraf

import (
	"fmt"
	"net"
	"strings"
	"testing"
	"time"
)

// mockTelegrafServer simulates a Telegraf server for testing
type mockTelegrafServer struct {
	listener     net.Listener
	receivedData []string
	failCount    int
	shouldFail   bool
}

func newMockTelegrafServer(t *testing.T) (*mockTelegrafServer, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}

	server := &mockTelegrafServer{
		listener:     listener,
		receivedData: make([]string, 0),
	}

	// Update the package variables to point to our mock server
	telegrafHost = "127.0.0.1"
	telegrafPort = fmt.Sprintf("%d", listener.Addr().(*net.TCPAddr).Port)

	// Reduce retry parameters for faster tests
	maxRetries = 3
	initialBackoff = 10 * time.Millisecond

	// Start the mock server
	go server.start(t)

	return server, nil
}

func (m *mockTelegrafServer) start(t *testing.T) {
	for {
		conn, err := m.listener.Accept()
		if err != nil {
			// If the listener was closed, just return
			if strings.Contains(err.Error(), "use of closed network connection") {
				return
			}
			t.Logf("Error accepting connection: %v", err)
			continue
		}

		go m.handleConnection(t, conn)
	}
}

func (m *mockTelegrafServer) handleConnection(t *testing.T, conn net.Conn) {
	defer conn.Close()

	buf := make([]byte, 1024)
	for {
		if m.shouldFail && m.failCount > 0 {
			m.failCount--
			conn.Close()
			return
		}

		n, err := conn.Read(buf)
		if err != nil {
			// If the connection was closed, just return
			if strings.Contains(err.Error(), "use of closed network connection") ||
			   strings.Contains(err.Error(), "EOF") {
				return
			}
			t.Logf("Error reading from connection: %v", err)
			return
		}

		data := string(buf[:n])
		lines := strings.Split(strings.TrimSpace(data), "\n")
		for _, line := range lines {
			if line != "" {
				m.receivedData = append(m.receivedData, line)
			}
		}
	}
}

func (m *mockTelegrafServer) close() {
	m.listener.Close()
}

func (m *mockTelegrafServer) setFailMode(shouldFail bool, failCount int) {
	m.shouldFail = shouldFail
	m.failCount = failCount
}

func TestSetupTelegrafConnection(t *testing.T) {
	// Start a mock server
	server, err := newMockTelegrafServer(t)
	if err != nil {
		t.Fatalf("Failed to start mock server: %v", err)
	}
	defer server.close()
	defer CloseTelegrafConnection()

	// Test successful connection
	SetupTelegrafConnection()
	if sharedConn == nil {
		t.Fatal("Expected connection to be established, but it's nil")
	}
}

func TestSendToTelegraf(t *testing.T) {
	// Start a mock server
	server, err := newMockTelegrafServer(t)
	if err != nil {
		t.Fatalf("Failed to start mock server: %v", err)
	}
	defer server.close()
	defer CloseTelegrafConnection()

	// Establish connection
	SetupTelegrafConnection()

	// Test sending data
	testData := []string{
		"cpu,host=server01 usage=0.5 1617459432000000000",
		"memory,host=server01 used_percent=65.23 1617459432000000000",
	}

	err = SendToTelegraf(testData)
	if err != nil {
		t.Fatalf("Failed to send data: %v", err)
	}

	// Give some time for the server to process the data
	time.Sleep(100 * time.Millisecond)

	// Verify the data was received
	if len(server.receivedData) != len(testData) {
		t.Fatalf("Expected %d data points, got %d", len(testData), len(server.receivedData))
	}

	for i, expected := range testData {
		if server.receivedData[i] != expected {
			t.Errorf("Expected data point %d to be '%s', got '%s'", i, expected, server.receivedData[i])
		}
	}
}

func TestSendToTelegrafWithRetries(t *testing.T) {
	// Start a mock server
	server, err := newMockTelegrafServer(t)
	if err != nil {
		t.Fatalf("Failed to start mock server: %v", err)
	}
	defer server.close()
	defer CloseTelegrafConnection()

	// Establish connection
	SetupTelegrafConnection()

	// Set the server to fail once
	server.setFailMode(true, 1)

	// Test sending data
	testData := []string{
		"cpu,host=server01 usage=0.5 1617459432000000000",
	}

	err = SendToTelegraf(testData)
	if err != nil {
		t.Fatalf("Failed to send data with retry: %v", err)
	}

	// Give some time for the server to process the data
	time.Sleep(200 * time.Millisecond)

	// Verify the data was received
	if len(server.receivedData) != len(testData) {
		t.Fatalf("Expected %d data points after retry, got %d", len(testData), len(server.receivedData))
	}
}

func TestSendToTelegrafWithNoConnection(t *testing.T) {
	// Reset the shared connection
	sharedConn = nil

	// Test sending data with no connection
	testData := []string{
		"cpu,host=server01 usage=0.5 1617459432000000000",
	}

	err := SendToTelegraf(testData)
	if err == nil {
		t.Fatal("Expected error when sending with no connection, but got nil")
	}

	if err.Error() != "Telegraf connection is not established" {
		t.Fatalf("Expected 'Telegraf connection is not established' error, got: %v", err)
	}
}

func TestIsValidLineProtocol(t *testing.T) {
	validLines := []string{
		"cpu,host=server01,region=us-west usage=0.5,idle=99.5 1617459432000000000",
		"memory,host=server01 used_percent=65.23 1617459432000000000",
	}

	invalidLines := []string{
		"invalid line",
		"cpu host=server01 usage=0.5", // Missing comma
		"cpu,host=server01 1617459432000000000", // Missing field
	}

	for _, line := range validLines {
		if !IsValidLineProtocol(line) {
			t.Errorf("Expected line to be valid: %s", line)
		}
	}

	for _, line := range invalidLines {
		if IsValidLineProtocol(line) {
			t.Errorf("Expected line to be invalid: %s", line)
		}
	}
}

func TestReconnectTelegraf(t *testing.T) {
	// Start a mock server
	server, err := newMockTelegrafServer(t)
	if err != nil {
		t.Fatalf("Failed to start mock server: %v", err)
	}
	defer server.close()

	// Establish initial connection
	SetupTelegrafConnection()
	defer CloseTelegrafConnection()

	// Close the connection to simulate a failure
	sharedConn.Close()
	sharedConn = nil

	// Test reconnection
	err = reconnectTelegraf()
	if err != nil {
		t.Fatalf("Failed to reconnect: %v", err)
	}

	if sharedConn == nil {
		t.Fatal("Expected connection to be re-established, but it's nil")
	}
}

func TestReconnectTelegrafFailure(t *testing.T) {
	// Set invalid host to force reconnection failure
	originalHost := telegrafHost
	telegrafHost = "nonexistent-host"
	defer func() { telegrafHost = originalHost }()

	// Test reconnection to invalid host
	err := reconnectTelegraf()
	if err == nil {
		t.Fatal("Expected reconnection to fail, but it succeeded")
	}
}
