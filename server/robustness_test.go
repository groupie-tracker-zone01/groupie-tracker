package server

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestFetchJSONTimeoutStopsSlowUpstream(t *testing.T) {
	started := make(chan struct{})
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		select {
		case <-r.Context().Done():
			return
		case <-time.After(500 * time.Millisecond):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{"id":1,"name":"Queen"}]`))
		}
	}))
	defer testServer.Close()

	client := &http.Client{Timeout: 50 * time.Millisecond}
	var target []Artist
	start := time.Now()
	err := fetchJSONContext(context.Background(), client, testServer.URL, &target)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("slow upstream should trigger a client timeout")
	}
	if elapsed > 300*time.Millisecond {
		t.Fatalf("slow upstream blocked for %v, expected a bounded request", elapsed)
	}
	select {
	case <-started:
	default:
		t.Fatal("upstream request never started")
	}
}

func TestFetchJSONContextCancellationStopsRequest(t *testing.T) {
	started := make(chan struct{})
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-r.Context().Done()
	}))
	defer testServer.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client := &http.Client{Timeout: time.Second}
	errCh := make(chan error, 1)
	go func() {
		var target []Artist
		errCh <- fetchJSONContext(ctx, client, testServer.URL, &target)
	}()

	select {
	case <-started:
		cancel()
	case <-time.After(300 * time.Millisecond):
		t.Fatal("upstream request did not start")
	}

	select {
	case err := <-errCh:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancelled request returned %v, want context.Canceled", err)
		}
	case <-time.After(300 * time.Millisecond):
		t.Fatal("cancelled request remained blocked")
	}
}

func TestFetchJSONRecoversAfterUpstreamErrors(t *testing.T) {
	var calls atomic.Int32
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) <= 5 {
			http.Error(w, "temporary upstream failure", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"id":1,"name":"Queen"}]`))
	}))
	defer testServer.Close()

	client := &http.Client{Timeout: time.Second}
	for i := 0; i < 5; i++ {
		var target []Artist
		if err := fetchJSONContext(context.Background(), client, testServer.URL, &target); err == nil {
			t.Fatalf("request %d should fail while upstream returns 503", i+1)
		}
	}

	var target []Artist
	if err := fetchJSONContext(context.Background(), client, testServer.URL, &target); err != nil {
		t.Fatalf("client did not recover after upstream errors: %v", err)
	}
	if len(target) != 1 || target[0].Name != "Queen" {
		t.Fatalf("unexpected recovery payload: %+v", target)
	}
}

func TestTemplateFailureRendersClean500Page(t *testing.T) {
	templates := template.Must(template.New("robustness").Parse(`
{{define "home"}}PARTIAL CONTENT {{template "missing" .}}{{end}}
{{define "500"}}SERVER ERROR PAGE{{end}}
`))
	handler := Routes(templates, &AppData{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	if res.Code != http.StatusInternalServerError {
		t.Fatalf("template failure returned %d, want %d", res.Code, http.StatusInternalServerError)
	}
	body := res.Body.String()
	if !strings.Contains(body, "SERVER ERROR PAGE") {
		t.Fatalf("500 page was not rendered: %q", body)
	}
	if strings.Contains(body, "PARTIAL CONTENT") {
		t.Fatalf("partial broken page leaked before 500 response: %q", body)
	}
}

func TestConcurrentRoutesRemainResponsive(t *testing.T) {
	data, fullArtists := testAppData()
	handler := Routes(testTemplates(t), data, fullArtists)

	const (
		workers    = 24
		perWorker  = 20
		maxLatency = 2 * time.Second
	)

	baseline := runtime.NumGoroutine()
	start := time.Now()
	var wg sync.WaitGroup
	errs := make(chan error, workers)

	for worker := 0; worker < workers; worker++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for i := 0; i < perWorker; i++ {
				path := "/"
				want := http.StatusOK
				switch (worker + i) % 4 {
				case 1:
					path = "/artists?q=Queen"
				case 2:
					path = "/api/artists"
				case 3:
					path = "/artists"
					want = http.StatusBadRequest
				}

				req := httptest.NewRequest(http.MethodGet, path, nil)
				res := httptest.NewRecorder()
				handler.ServeHTTP(res, req)
				if res.Code != want {
					errs <- fmt.Errorf("GET %s returned %d, want %d", path, res.Code, want)
					return
				}
			}
		}(worker)
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(maxLatency):
		t.Fatal("concurrent request burst blocked or froze")
	}
	close(errs)
	for err := range errs {
		t.Error(err)
	}

	if elapsed := time.Since(start); elapsed > maxLatency {
		t.Fatalf("concurrent burst took %v, want less than %v", elapsed, maxLatency)
	}

	runtime.GC()
	time.Sleep(20 * time.Millisecond)
	if after := runtime.NumGoroutine(); after > baseline+8 {
		t.Fatalf("goroutine count grew abnormally: before=%d after=%d", baseline, after)
	}

	res := httptest.NewRecorder()
	handler.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("server did not remain available after burst: got %d", res.Code)
	}
}

func TestCancelledClientRequestDoesNotBlockHandler(t *testing.T) {
	data, fullArtists := testAppData()
	handler := Routes(testTemplates(t), data, fullArtists)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req := httptest.NewRequest(http.MethodGet, "/artists?q=Queen", nil).WithContext(ctx)
	res := httptest.NewRecorder()
	done := make(chan struct{})

	go func() {
		handler.ServeHTTP(res, req)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(300 * time.Millisecond):
		t.Fatal("handler remained blocked after client cancellation")
	}
}

func benchmarkTemplates() *template.Template {
	return template.Must(template.New("benchmark").Parse(`
{{define "home"}}HOME{{end}}
{{define "artists"}}{{range .Artists}}{{.Name}}{{end}}{{end}}
{{define "400"}}BAD REQUEST{{end}}
{{define "403"}}FORBIDDEN{{end}}
{{define "404"}}NOT FOUND{{end}}
{{define "405"}}METHOD NOT ALLOWED{{end}}
{{define "500"}}SERVER ERROR{{end}}
`))
}

func BenchmarkRoutesLightBurst(b *testing.B) {
	data := &AppData{}
	fullArtists := []ArtistFull{{Id: 1, Name: "Queen"}, {Id: 2, Name: "Queens of the Stone Age"}}
	handler := Routes(benchmarkTemplates(), data, fullArtists)

	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest(http.MethodGet, "/artists?q=Que", nil)
			res := httptest.NewRecorder()
			handler.ServeHTTP(res, req)
			if res.Code != http.StatusOK {
				b.Fatalf("GET /artists returned %d", res.Code)
			}
		}
	})
}
