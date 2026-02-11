// pkg/portfwd/forwarder_test.go

package portfwd

import (
	"net"
	"testing"
)

func TestCheckPortAvailable_Free(t *testing.T) {
	// Find a free port
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()

	// Port should be available now
	err = checkPortAvailable(int32(port))
	if err != nil {
		t.Errorf("port should be available: %v", err)
	}
}

func TestCheckPortAvailable_InUse(t *testing.T) {
	// Occupy a port
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	port := ln.Addr().(*net.TCPAddr).Port

	// Port should NOT be available
	err = checkPortAvailable(int32(port))
	if err == nil {
		t.Error("port should NOT be available")
	}
}

func TestSuggestAlternativePort(t *testing.T) {
	// Occupy a port
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	occupiedPort := int32(ln.Addr().(*net.TCPAddr).Port)

	// Should find an alternative
	alt, err := SuggestAlternativePort(occupiedPort)
	if err != nil {
		t.Fatalf("SuggestAlternativePort failed: %v", err)
	}

	if alt == occupiedPort {
		t.Error("should suggest different port")
	}

	// Alternative should be available
	if err := checkPortAvailable(alt); err != nil {
		t.Errorf("suggested port %d not available: %v", alt, err)
	}
}

func TestSuggestAlternativePort_PreferredAvailable(t *testing.T) {
	// Find a free port
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	port := int32(ln.Addr().(*net.TCPAddr).Port)
	ln.Close()

	// Should return preferred port if available
	alt, err := SuggestAlternativePort(port)
	if err != nil {
		t.Fatalf("SuggestAlternativePort failed: %v", err)
	}

	if alt != port {
		t.Errorf("should return preferred port, got %d", alt)
	}
}
