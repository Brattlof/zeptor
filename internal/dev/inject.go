package dev

import (
	"bytes"
	"net/http"
)

const hmrClientScript = `<script>
(function() {
  var ws;
  var reconnectAttempts = 0;
  var maxReconnectDelay = 5000;

  function connect() {
    var protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    ws = new WebSocket(protocol + '//' + window.location.host + '/__hmr');

    ws.onopen = function() {
      console.log('[HMR] Connected');
      reconnectAttempts = 0;
    };

    ws.onmessage = function(e) {
      var msg = JSON.parse(e.data);
      if (msg.type === 'reload') {
        console.log('[HMR] Reloading:', msg.file);
        window.location.reload();
      }
    };

    ws.onclose = function() {
      var delay = Math.min(1000 * Math.pow(2, reconnectAttempts), maxReconnectDelay);
      reconnectAttempts++;
      console.log('[HMR] Disconnected, reconnecting in ' + delay + 'ms');
      setTimeout(connect, delay);
    };

    ws.onerror = function(err) {
      console.error('[HMR] Error:', err);
    };
  }

  connect();
})();
</script>`

type hmrResponseWriter struct {
	http.ResponseWriter
	buf        *bytes.Buffer
	statusCode int
	injectJS   bool
}

func (w *hmrResponseWriter) Write(b []byte) (int, error) {
	if !w.injectJS {
		return w.ResponseWriter.Write(b)
	}
	return w.buf.Write(b)
}

func (w *hmrResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	if !w.injectJS {
		w.ResponseWriter.WriteHeader(statusCode)
	}
}

func (w *hmrResponseWriter) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func InjectHMR(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/__hmr" {
			next.ServeHTTP(w, r)
			return
		}

		if r.Header.Get("Upgrade") == "websocket" {
			next.ServeHTTP(w, r)
			return
		}

		hw := &hmrResponseWriter{
			ResponseWriter: w,
			buf:            &bytes.Buffer{},
			statusCode:     http.StatusOK,
			injectJS:       true,
		}

		next.ServeHTTP(hw, r)

		body := hw.buf.Bytes()
		if len(body) == 0 {
			return
		}

		if !bytes.Contains(bytes.ToLower(body[:min(1024, len(body))]), []byte("<!doctype")) &&
			!bytes.Contains(bytes.ToLower(body[:min(1024, len(body))]), []byte("<html")) {
			w.Write(body)
			return
		}

		idx := bytes.LastIndex(body, []byte("</body>"))
		if idx == -1 {
			idx = bytes.LastIndex(body, []byte("</html>"))
		}
		if idx == -1 {
			w.Write(body)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(hw.statusCode)

		w.Write(body[:idx])
		w.Write([]byte(hmrClientScript))
		w.Write(body[idx:])
	})
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
