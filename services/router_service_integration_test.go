package services

import (
	"context"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Agushim/go_wifi_billing/lib"
)

func TestMikrotikRouterOSConnectionFlow(t *testing.T) {
	host := strings.TrimSpace(os.Getenv("MIKROTIK_TEST_HOST"))
	username := strings.TrimSpace(os.Getenv("MIKROTIK_TEST_USERNAME"))
	password := os.Getenv("MIKROTIK_TEST_PASSWORD")
	if host == "" || username == "" || password == "" {
		t.Skip("set MIKROTIK_TEST_HOST, MIKROTIK_TEST_USERNAME, and MIKROTIK_TEST_PASSWORD to run RouterOS integration test")
	}

	port := 8728
	if raw := strings.TrimSpace(os.Getenv("MIKROTIK_TEST_PORT")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			t.Fatalf("invalid MIKROTIK_TEST_PORT: %v", err)
		}
		port = parsed
	}

	useTLS := parseBoolEnv("MIKROTIK_TEST_USE_TLS")
	connectTimeout := 5 * time.Second
	commandTimeout := 5 * time.Second
	if raw := strings.TrimSpace(os.Getenv("MIKROTIK_TEST_CONNECT_TIMEOUT")); raw != "" {
		parsed, err := time.ParseDuration(raw)
		if err != nil {
			t.Fatalf("invalid MIKROTIK_TEST_CONNECT_TIMEOUT: %v", err)
		}
		connectTimeout = parsed
	}
	if raw := strings.TrimSpace(os.Getenv("MIKROTIK_TEST_COMMAND_TIMEOUT")); raw != "" {
		parsed, err := time.ParseDuration(raw)
		if err != nil {
			t.Fatalf("invalid MIKROTIK_TEST_COMMAND_TIMEOUT: %v", err)
		}
		commandTimeout = parsed
	}

	runner, err := lib.NewMikrotikRunner(host, port, useTLS, connectTimeout, commandTimeout)
	if err != nil {
		t.Fatalf("create runner: %v", err)
	}
	defer runner.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	login := runner.Login(ctx, username, password)
	if !login.Success {
		t.Fatalf("login failed: code=%s err=%v", login.ErrorCode, login.Err)
	}

	identity := runner.RunReadOnly(ctx, 2, "/system/identity/print")
	if !identity.Success {
		t.Fatalf("identity command failed: code=%s err=%v", identity.ErrorCode, identity.Err)
	}
	if len(identity.Items) == 0 || strings.TrimSpace(identity.Items[0]["name"]) == "" {
		t.Fatalf("identity response is empty")
	}

	resource := runner.RunReadOnly(ctx, 2, "/system/resource/print")
	if !resource.Success {
		t.Fatalf("resource command failed: code=%s err=%v", resource.ErrorCode, resource.Err)
	}
	if len(resource.Items) == 0 {
		t.Fatalf("resource response is empty")
	}

	if strings.TrimSpace(resource.Items[0]["version"]) == "" {
		t.Fatalf("routeros version is empty")
	}
}

func parseBoolEnv(name string) bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(name)))
	switch value {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}
