package main

import (
	"net/http"
	"testing"
)

func TestServerPort(t *testing.T) {
	t.Run("default port", func(t *testing.T) {
		t.Setenv(portEnv, "")
		if got := serverPort(); got != defaultPort {
			t.Fatalf("serverPort() = %q, want %q", got, defaultPort)
		}
	})

	t.Run("environment port", func(t *testing.T) {
		t.Setenv(portEnv, "9090")
		if got := serverPort(); got != "9090" {
			t.Fatalf("serverPort() = %q, want %q", got, "9090")
		}
	})
}

func TestHTTPServerTimeouts(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	serverHTTP := newHTTPServer("9090", handler)
	if serverHTTP.Addr != ":9090" {
		t.Fatalf("server address = %q, want %q", serverHTTP.Addr, ":9090")
	}
	if serverHTTP.ReadHeaderTimeout != serverReadHeaderTimeout {
		t.Fatalf("ReadHeaderTimeout = %v, want %v", serverHTTP.ReadHeaderTimeout, serverReadHeaderTimeout)
	}
	if serverHTTP.ReadTimeout != serverReadTimeout {
		t.Fatalf("ReadTimeout = %v, want %v", serverHTTP.ReadTimeout, serverReadTimeout)
	}
	if serverHTTP.WriteTimeout != serverWriteTimeout {
		t.Fatalf("WriteTimeout = %v, want %v", serverHTTP.WriteTimeout, serverWriteTimeout)
	}
	if serverHTTP.IdleTimeout != serverIdleTimeout {
		t.Fatalf("IdleTimeout = %v, want %v", serverHTTP.IdleTimeout, serverIdleTimeout)
	}
}
