package main

import "testing"

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
