package wikipedia

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func Test_Client_Success(t *testing.T) {
	ms := runMockServer()
	defer ms.Stop()

	html, err := SourceHTML(context.Background(), fmt.Sprintf("%v/page", ms.Address()))

	if err != nil {
		t.Fatalf("Unexpected error sourcing HMTL: %v", err)
	}

	if html == nil {
		t.Fatalf("Unexpected null HMTL")
	}
}

func Test_Client_Fail(t *testing.T) {
	ms := runMockServer()
	defer ms.Stop()

	var scenario = []struct {
		name             string
		url              string
		timeout          time.Duration
		expectedContains string
	}{
		{
			name:             "Unreacheable URL",
			url:              "http://123",
			timeout:          1 * time.Second,
			expectedContains: "deadline exceeded",
		},
		{
			name:             "Invalid URL",
			url:              ms.Address() + "/page1",
			timeout:          6 * time.Second,
			expectedContains: "404",
		},
	}

	for _, s := range scenario {
		t.Run(s.name, func(t *testing.T) {
			_, err := SourceHTML(context.Background(), s.url)

			if !strings.Contains(err.Error(), s.expectedContains) {
				t.Fatalf("Expected error, got none, %v", err)
			}
		})
	}
}

type mockServer struct {
	s *httptest.Server
}

func (m *mockServer) Address() string {
	return m.s.URL
}

func (m *mockServer) Stop() {
	m.s.Close()
}

func runMockServer() *mockServer {
	file, err := os.ReadFile("../testdata/httpstatus_page.html")
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/page", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write(file)
	})

	return &mockServer{
		s: httptest.NewServer(mux),
	}
}
