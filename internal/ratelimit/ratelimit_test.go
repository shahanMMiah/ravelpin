package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func limitMiddleware(next http.Handler, rl *RateLimiter) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr

		if !rl.GetClientRateLimit(ip) {
			http.Error(w, "Too Many Requests for i", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func testHandler() http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		resp.WriteHeader(200)
		resp.Header().Set("Content-Type", "text/plain; charset=utf-8")
		resp.Write([]byte("hit test endpoint!"))

	})
}

func TestRateLimit(t *testing.T) {

	rl := NewRateLimiter()

	handler := limitMiddleware(testHandler(), rl)
	server := httptest.NewServer(handler)

	for range BURST + 1 {

		req := httptest.NewRequest("GET", server.URL, nil)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Result().StatusCode != 200 {
			t.Errorf("response should be 200, rl tokens left %v", rl.GetTokenAmount(req.RemoteAddr))
		}

	}

	req := httptest.NewRequest("GET", server.URL, nil)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Result().StatusCode != http.StatusTooManyRequests {
		t.Errorf("response should be 429, but is %v rl tokens left %v ip %v", rr.Result().StatusCode, rl.GetTokenAmount(req.RemoteAddr), req.RemoteAddr)
	}

}
