package wc

import (
	"os"
	"testing"
	"time"
)

func requireEnv(t *testing.T, key string) string {
	t.Helper()
	v := os.Getenv(key)
	if v == "" {
		t.Skipf("skip integration test: missing env %s", key)
	}
	return v
}

func TestIntegrationNewAndClose(t *testing.T) {
	appID := requireEnv(t, "WC_APP_ID")
	appKey := requireEnv(t, "WC_APP_KEY")
	agentID := requireEnv(t, "WC_AGENT_ID")

	client, err := New(appID, appKey, agentID)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	client.Close()
}

func TestIntegrationSend(t *testing.T) {
	appID := requireEnv(t, "WC_APP_ID")
	appKey := requireEnv(t, "WC_APP_KEY")
	agentID := requireEnv(t, "WC_AGENT_ID")
	toUser := requireEnv(t, "WC_TO_USER")

	client, err := New(appID, appKey, agentID)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer client.Close()

	msg := "wc integration test @ " + time.Now().Format(time.RFC3339)
	if err := client.Send(toUser, msg); err != nil {
		t.Fatalf("Send() failed: %v", err)
	}
}
