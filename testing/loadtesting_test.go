package loadtesting

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/hannahkm/gopherconus-2025/handlers"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func TestHelloEndpoint(t *testing.T) {
	server := SetupTestingEndpoints()
	defer server.Close()

	resp, err := http.Get(server.URL + "/hello")
	if err != nil {
		t.Fatalf("GET request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected to receive status OK, got: %s", resp.Status)
	}

	if content := resp.Header.Get("Content-Type"); content != "application/json" {
		t.Fatalf("Expected application/json content type, got %v", content)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Error reading response body: %v", err)
	}

	var response handlers.HelloResponse
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("Error parsing response JSON: %v", err)
	}

	if v := response.Message; v != "Hello World!" {
		t.Fatalf("Expected 'Hello World!', got %v", v)
	}
}
