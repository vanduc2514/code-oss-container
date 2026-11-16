// Command nginx-routes turns the remoteAgentHosts entries of config.json into
// nginx locations, so the browser only ever connects to the Code - OSS origin
// and nginx makes the hop into the private network.
//
// It runs as a /docker-entrypoint.d script: nginx has not started yet, and a
// non-zero exit aborts startup instead of serving a broken config.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
)

type host struct {
	Name            string `json:"name"`
	Path            string `json:"path"`
	Address         string `json:"address"`
	Upstream        string `json:"upstream"`
	ConnectionToken string `json:"connectionToken"`
}

type config struct {
	RemoteAgentHosts []host `json:"remoteAgentHosts"`
}

type route struct {
	Path     string
	Upstream string
}

var (
	pathPattern     = regexp.MustCompile(`^/ahp(/[A-Za-z0-9._-]+)*$`)
	upstreamPattern = regexp.MustCompile(`^https?://[A-Za-z0-9._-]+(:[0-9]+)?(/[A-Za-z0-9._~%/-]*)?$`)
	slugSeparators  = regexp.MustCompile(`[^a-z0-9._-]+`)
)

// routeTemplate emits a self-contained location block: everything nginx needs
// to bridge a browser WebSocket to a private agent host.
const routeTemplate = `location = {{.Path}} {
	access_log off;
	proxy_pass {{.Upstream}};
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

// pathFor returns the same-origin path for a host: its explicit path, or a
// slug of its name. Both end up in an nginx config file, so callers validate.
func pathFor(h host) string {
	if h.Path != "" {
		return h.Path
	}
	slug := slugSeparators.ReplaceAllString(strings.ToLower(h.Name), "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		return ""
	}
	return "/ahp/" + slug
}

// plan selects the hosts that need proxying and validates every value that
// reaches the generated nginx config.
func plan(cfg config) ([]route, error) {
	seen := map[string]string{}
	routes := make([]route, 0, len(cfg.RemoteAgentHosts))

	for i, h := range cfg.RemoteAgentHosts {
		if h.Upstream == "" {
			continue
		}
		label := h.Name
		if label == "" {
			label = fmt.Sprintf("host %d", i+1)
		}

		path := pathFor(h)
		if path == "" {
			return nil, fmt.Errorf("%s: needs a path or a name usable as one", label)
		}
		if !pathPattern.MatchString(path) {
			return nil, fmt.Errorf("%s: invalid path %q, expected /ahp or /ahp/<segment>", label, path)
		}
		if other, dup := seen[path]; dup {
			return nil, fmt.Errorf("%s: path %q is already used by %s", label, path, other)
		}
		if !upstreamPattern.MatchString(h.Upstream) {
			return nil, fmt.Errorf("%s: invalid upstream %q, expected http(s)://host[:port][/path]", label, h.Upstream)
		}

		seen[path] = label
		routes = append(routes, route{Path: path, Upstream: h.Upstream})
	}

	return routes, nil
}

func render(routes []route) (string, error) {
	tpl, err := template.New("routes").Parse(routeTemplate)
	if err != nil {
		return "", err
	}

	var out strings.Builder
	for _, r := range routes {
		if err := tpl.Execute(&out, r); err != nil {
			return "", err
		}
	}
	return out.String(), nil
}

// writeFileAtomic keeps nginx from ever reading a half-written include.
func writeFileAtomic(path, content string) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o644); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func run(configPath, outPath string) error {
	content, err := os.ReadFile(configPath)
	if errors.Is(err, os.ErrNotExist) {
		fmt.Printf("[nginx-routes] no %s; no proxy routes generated\n", configPath)
		return writeFileAtomic(outPath, "")
	}
	if err != nil {
		return err
	}

	var cfg config
	if err := json.Unmarshal(content, &cfg); err != nil {
		return fmt.Errorf("%s: %w", configPath, err)
	}

	routes, err := plan(cfg)
	if err != nil {
		return fmt.Errorf("%s: %w", configPath, err)
	}

	rendered, err := render(routes)
	if err != nil {
		return err
	}
	if err := writeFileAtomic(outPath, rendered); err != nil {
		return err
	}

	for _, r := range routes {
		fmt.Printf("[nginx-routes] %s -> %s\n", r.Path, r.Upstream)
	}
	fmt.Printf("[nginx-routes] wrote %s (%d routes)\n", outPath, len(routes))
	return nil
}

func main() {
	configPath := flag.String("config", envOr("AHP_CONFIG", "/usr/share/nginx/html/config.json"), "path to config.json")
	outPath := flag.String("out", envOr("AHP_ROUTES", "/etc/nginx/ahp/routes.conf"), "path to the generated nginx include")
	flag.Parse()

	if err := run(*configPath, *outPath); err != nil {
		fmt.Fprintf(os.Stderr, "[nginx-routes] %v\n", err)
		os.Exit(1)
	}
}
