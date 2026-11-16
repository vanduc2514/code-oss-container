package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPlan(t *testing.T) {
	tests := []struct {
		name    string
		hosts   []host
		want    []route
		wantErr string
	}{
		{
			name:  "explicit path",
			hosts: []host{{Name: "team-a", Path: "/ahp/team-a", Upstream: "http://gw-a:8765/ahp"}},
			want:  []route{{Path: "/ahp/team-a", Upstream: "http://gw-a:8765/ahp"}},
		},
		{
			name:  "path from name",
			hosts: []host{{Name: "Team A (prod)", Upstream: "http://gw-a:8765/ahp"}},
			want:  []route{{Path: "/ahp/team-a-prod", Upstream: "http://gw-a:8765/ahp"}},
		},
		{
			name: "host without upstream is skipped",
			hosts: []host{
				{Name: "direct", Address: "wss://agents.example.com/ahp"},
				{Name: "proxied", Upstream: "http://gw:8765/ahp"},
			},
			want: []route{{Path: "/ahp/proxied", Upstream: "http://gw:8765/ahp"}},
		},
		{
			name:  "root path",
			hosts: []host{{Name: "gateway", Path: "/ahp", Upstream: "https://gw.internal/ahp"}},
			want:  []route{{Path: "/ahp", Upstream: "https://gw.internal/ahp"}},
		},
		{
			name:    "path outside /ahp",
			hosts:   []host{{Name: "bad", Path: "/proxy", Upstream: "http://gw:8765/ahp"}},
			wantErr: "invalid path",
		},
		{
			name:    "directive injection in path",
			hosts:   []host{{Name: "bad", Path: "/ahp/x;return 200", Upstream: "http://gw:8765/ahp"}},
			wantErr: "invalid path",
		},
		{
			name:    "newline injection in upstream",
			hosts:   []host{{Name: "bad", Path: "/ahp/x", Upstream: "http://gw/ahp;\ninclude /etc/passwd"}},
			wantErr: "invalid upstream",
		},
		{
			name:    "unsupported upstream scheme",
			hosts:   []host{{Name: "bad", Path: "/ahp/x", Upstream: "file:///etc/passwd"}},
			wantErr: "invalid upstream",
		},
		{
			name:    "unusable name",
			hosts:   []host{{Name: "***", Upstream: "http://gw:8765/ahp"}},
			wantErr: "needs a path",
		},
		{
			name: "duplicate path",
			hosts: []host{
				{Name: "a", Path: "/ahp/x", Upstream: "http://gw-a:8765/ahp"},
				{Name: "b", Path: "/ahp/x", Upstream: "http://gw-b:8765/ahp"},
			},
			wantErr: "already used",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := plan(config{RemoteAgentHosts: tc.hosts})
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("got %d routes, want %d: %v", len(got), len(tc.want), got)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("route %d = %+v, want %+v", i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestRender(t *testing.T) {
	rendered, err := render([]route{{Path: "/ahp/team-a", Upstream: "http://gw-a:8765/ahp"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := `location = /ahp/team-a {
	access_log off;
	proxy_pass http://gw-a:8765/ahp;
	proxy_http_version 1.1;
	proxy_set_header Upgrade $http_upgrade;
	proxy_set_header Connection "upgrade";
	proxy_set_header Host $host;
	proxy_set_header X-Real-IP $remote_addr;
	proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
	proxy_set_header X-Forwarded-Proto $scheme;
	proxy_buffering off;
	proxy_read_timeout 3600s;
	proxy_send_timeout 3600s;
	proxy_ssl_server_name on;
}

`
	if rendered != want {
		t.Errorf("rendered =\n%s\nwant\n%s", rendered, want)
	}
}

func TestRun(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	outPath := filepath.Join(dir, "routes.conf")

	// Missing config writes an empty include and succeeds.
	if err := run(configPath, outPath); err != nil {
		t.Fatalf("missing config: %v", err)
	}
	if got, _ := os.ReadFile(outPath); len(got) != 0 {
		t.Errorf("expected empty include, got %q", got)
	}

	// Valid config produces one location per proxied host.
	write := func(body string) {
		t.Helper()
		if err := os.WriteFile(configPath, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(`{"remoteAgentHosts":[
		{"name":"direct","address":"wss://example.invalid/ahp"},
		{"name":"gateway","path":"/ahp","upstream":"http://gw:8765/ahp","connectionToken":"secret"}
	]}`)
	if err := run(configPath, outPath); err != nil {
		t.Fatalf("valid config: %v", err)
	}
	got, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "location = /ahp {") ||
		!strings.Contains(string(got), "proxy_pass http://gw:8765/ahp;") {
		t.Errorf("unexpected include:\n%s", got)
	}
	if strings.Contains(string(got), "secret") {
		t.Errorf("connection token leaked into nginx config:\n%s", got)
	}

	// Invalid JSON and invalid values fail.
	write(`not json`)
	if err := run(configPath, outPath); err == nil {
		t.Error("expected error for invalid JSON")
	}
	write(`{"remoteAgentHosts":[{"name":"bad","path":"/ahp/x","upstream":"http://gw/ahp;include x"}]}`)
	if err := run(configPath, outPath); err == nil {
		t.Error("expected error for invalid upstream")
	}
}
