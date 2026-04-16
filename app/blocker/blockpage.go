package blocker

/*
 * blockpage.go — HTTP server that shows "ACCESS DENIED" for blocked domains.
 *
 * CRITICAL from design spec: This server MUST start BEFORE the hosts
 * file is modified. Otherwise blocked domains show "connection refused"
 * instead of our helpful block page.
 *
 * The server listens on 127.0.0.1:9512 (configurable port).
 * When the hosts file redirects blocked domains to 127.0.0.1,
 * browsers on port 80 won't reach our server on 9512. But the hosts
 * file entries point to 127.0.0.1 and the browser defaults to port 80.
 *
 * SOLUTION: We listen on port 80 as well. This requires elevated
 * privileges on macOS/Linux (ports < 1024 are restricted).
 * Fallback: if port 80 fails, we listen on 9512 only.
 *
 * The block page is a self-contained HTML page with the terminal aesthetic.
 * It reads the Host header to know which domain was blocked.
 */

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
)

// BlockPageServer serves the "ACCESS DENIED" page for blocked domains.
type BlockPageServer struct {
	workspaceName string
	taskDesc      string
	timeRemaining string

	mu      sync.Mutex
	servers []*http.Server
}

// NewBlockPageServer creates a block page server.
func NewBlockPageServer(workspaceName, taskDesc string) *BlockPageServer {
	return &BlockPageServer{
		workspaceName: workspaceName,
		taskDesc:      taskDesc,
	}
}

// SetTimeRemaining updates the countdown shown on the block page.
func (bp *BlockPageServer) SetTimeRemaining(remaining string) {
	bp.mu.Lock()
	bp.timeRemaining = remaining
	bp.mu.Unlock()
}

// Start begins serving the block page. Tries port 80 first, falls back to 9512.
func (bp *BlockPageServer) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", bp.handleBlockPage)

	// Try to listen on port 80 (needs elevated privileges)
	if listener, err := net.Listen("tcp", "127.0.0.1:80"); err == nil {
		srv := &http.Server{Handler: mux}
		bp.mu.Lock()
		bp.servers = append(bp.servers, srv)
		bp.mu.Unlock()
		go srv.Serve(listener)
	}

	// Also listen on 9512 as a fallback
	srv := &http.Server{
		Addr:    "127.0.0.1:9512",
		Handler: mux,
	}
	bp.mu.Lock()
	bp.servers = append(bp.servers, srv)
	bp.mu.Unlock()

	go func() {
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			fmt.Printf("block page server error: %v\n", err)
		}
	}()

	return nil
}

// Stop shuts down all block page servers.
func (bp *BlockPageServer) Stop() {
	bp.mu.Lock()
	servers := bp.servers
	bp.servers = nil
	bp.mu.Unlock()

	for _, srv := range servers {
		srv.Shutdown(context.Background())
	}
}

// handleBlockPage renders the ACCESS DENIED page.
func (bp *BlockPageServer) handleBlockPage(w http.ResponseWriter, r *http.Request) {
	// The Host header tells us which domain the browser tried to reach
	blockedDomain := r.Host
	if idx := strings.Index(blockedDomain, ":"); idx >= 0 {
		blockedDomain = blockedDomain[:idx]
	}

	bp.mu.Lock()
	remaining := bp.timeRemaining
	workspace := bp.workspaceName
	task := bp.taskDesc
	bp.mu.Unlock()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusForbidden)

	// Self-contained HTML with the terminal aesthetic
	fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<title>silo — blocked</title>
<style>
  * { margin: 0; padding: 0; box-sizing: border-box; }
  body {
    background: #1a1d23;
    color: #c9d1d9;
    font-family: 'JetBrains Mono', 'Fira Code', 'SF Mono', monospace;
    font-size: 14px;
    display: flex;
    justify-content: center;
    align-items: center;
    min-height: 100vh;
    padding: 32px;
  }
  .box {
    border: 1px solid #30363d;
    max-width: 520px;
    width: 100%%;
    padding: 48px 40px;
    text-align: center;
  }
  .title {
    color: #f85149;
    font-size: 18px;
    font-weight: 700;
    margin-bottom: 24px;
    letter-spacing: 0.1em;
  }
  .domain {
    color: #f0f6fc;
    margin-bottom: 8px;
  }
  .meta {
    color: #484f58;
    font-size: 12px;
    margin-bottom: 4px;
  }
  .quote {
    color: #484f58;
    font-style: italic;
    margin: 24px 0;
  }
  .task {
    color: #7ee787;
    margin-bottom: 24px;
  }
  .hint {
    color: #484f58;
    font-size: 12px;
  }
</style>
</head>
<body>
<div class="box">
  <div class="title">ACCESS DENIED</div>
  <div class="domain">%s is not in your workspace</div>
  <div class="meta">workspace: %s</div>
  <div class="meta">remaining: %s</div>
  <div class="quote">"that's not what you're doing right now."</div>
  <div class="task">task: %s</div>
  <div class="hint">need this site? press [x] in silo</div>
</div>
</body>
</html>`, blockedDomain, workspace, remaining, task)
}
