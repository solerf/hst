package wikipedia

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func Test_SourceHTML_Success(t *testing.T) {
	ms := runMockServer()
	defer ms.Stop()

	node, err := SourceHTML(context.Background(), fmt.Sprintf("%v/page", ms.Address()))
	if err != nil {
		t.Fatalf("Unexpected error sourcing HTML: %v", err)
	}
	if node == nil {
		t.Fatalf("Unexpected null HTML")
	}
}

func Test_SourceRevision_Success(t *testing.T) {
	ms := runMockServer()
	defer ms.Stop()

	rev, err := SourceRevision(context.Background(), fmt.Sprintf("%v/bare", ms.Address()))
	if err != nil {
		t.Fatalf("Unexpected error sourcing revision: %v", err)
	}
	if rev != 1353518235 {
		t.Fatalf("Unexpected revision id: %v", rev)
	}
}

func Test_SourceHTML_Fail(t *testing.T) {
	ms := runMockServer()
	defer ms.Stop()

	hang := newHangingServer()
	defer hang.Close()

	scenario := []struct {
		name             string
		url              string
		timeout          time.Duration
		expectedContains string
	}{
		{
			name:             "Slow server beyond timeout",
			url:              hang.URL,
			timeout:          100 * time.Millisecond,
			expectedContains: "deadline exceeded",
		},
		{
			name:             "404 from server",
			url:              ms.Address() + "/missing",
			timeout:          6 * time.Second,
			expectedContains: "404",
		},
	}

	for _, s := range scenario {
		t.Run(s.name, func(t *testing.T) {
			ctx, cancelFunc := context.WithTimeout(context.Background(), s.timeout)
			defer cancelFunc()

			_, err := SourceHTML(ctx, s.url)
			if err == nil {
				t.Fatalf("Expected error, got none")
			}
			if !strings.Contains(err.Error(), s.expectedContains) {
				t.Fatalf("Expected error containing %q, got: %v", s.expectedContains, err)
			}
		})
	}
}

func Test_SourceRevision_Fail_BadJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `not-valid-json`)
	}))
	defer srv.Close()

	_, err := SourceRevision(context.Background(), srv.URL)
	if err == nil {
		t.Fatal("expected decode error")
	}
	if !strings.Contains(err.Error(), "decode revision") {
		t.Fatalf("expected decode-revision wrap, got: %v", err)
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
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(file)
	})
	mux.HandleFunc("/bare", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"latest":{"id":1353518235,"timestamp":"2026-05-10T18:55:50Z"}}`)
	})

	return &mockServer{
		s: httptest.NewServer(mux),
	}
}

// newHangingServer returns a server that holds the connection open
// until the client's context is cancelled — used to make timeout
// scenarios deterministic instead of depending on DNS / resolver speed.
func newHangingServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
}
