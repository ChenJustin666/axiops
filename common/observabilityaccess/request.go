package observabilityaccess

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

var idPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,127}$`)
var namespacePattern = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]{0,61}[a-z0-9])?$`)

func identifier(s string) bool { return idPattern.MatchString(s) }
func namespace(s string) bool  { return namespacePattern.MatchString(s) }

type requestCheck struct {
	target            Target
	workspaceID       uint64
	resource, action  string
	aggregate, global bool
}

// Read one canonical representation. Duplicate/case-aliased JSON keys and
// query/body collisions would otherwise be interpreted differently by parsers.
func requestFields(r *http.Request) (map[string]json.RawMessage, error) {
	fields := make(map[string]json.RawMessage)
	q, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		return nil, ErrDenied
	}
	for key, values := range q {
		if len(values) != 1 {
			return nil, ErrDenied
		}
		lower := strings.ToLower(key)
		for _, canonical := range []string{"clusterUuid", "namespace", "namespaces", "workloadId", "workspaceId", "projectUuid", "mode", "expr", "command", "queryMode", "sourceType", "logType", "logSourceType"} {
			if lower == strings.ToLower(canonical) && key != canonical {
				return nil, ErrDenied
			}
		}
		if _, found := fields[lower]; found {
			return nil, ErrDenied
		}
		fields[lower], _ = json.Marshal(values[0])
	}
	if r.Body == nil || r.Body == http.NoBody {
		return fields, nil
	}
	data, err := io.ReadAll(io.LimitReader(r.Body, (2<<20)+1))
	_ = r.Body.Close()
	r.Body = io.NopCloser(bytes.NewReader(data))
	if err != nil || len(data) > 2<<20 {
		return nil, ErrDenied
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return fields, nil
	}
	if r.Method == http.MethodGet {
		return nil, ErrDenied
	}
	// These observability POST endpoints consume JSON only.
	if !strings.Contains(strings.ToLower(r.Header.Get("Content-Type")), "application/json") {
		return nil, ErrDenied
	}
	d := json.NewDecoder(bytes.NewReader(data))
	token, err := d.Token()
	if err != nil || token != json.Delim('{') {
		return nil, ErrDenied
	}
	for d.More() {
		token, err := d.Token()
		if err != nil {
			return nil, ErrDenied
		}
		key, ok := token.(string)
		if !ok {
			return nil, ErrDenied
		}
		lower := strings.ToLower(key)
		if _, exists := fields[lower]; exists {
			return nil, ErrDenied
		}
		var value json.RawMessage
		if d.Decode(&value) != nil {
			return nil, ErrDenied
		}
		fields[lower] = value
	}
	if _, err := d.Token(); err != nil {
		return nil, ErrDenied
	}
	if _, err := d.Token(); err != io.EOF {
		return nil, ErrDenied
	}
	return fields, nil
}

func field(f map[string]json.RawMessage, key string) string {
	var s string
	if raw, ok := f[strings.ToLower(key)]; ok && json.Unmarshal(raw, &s) != nil {
		return "\x00"
	}
	return s
}

func uintField(f map[string]json.RawMessage, key string) string {
	raw, exists := f[strings.ToLower(key)]
	if !exists {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var n uint64
	if json.Unmarshal(raw, &n) == nil {
		return strconv.FormatUint(n, 10)
	}
	return "\x00"
}

func (g *Guard) requestChecks(r *http.Request) ([]requestCheck, bool, error) {
	p := r.URL.Path
	monitor := strings.HasPrefix(p, "/console/v1/cluster-monitor/") || strings.HasPrefix(p, "/console/v1/node-monitor/") || strings.HasPrefix(p, "/console/v1/namespace-monitor/") || strings.HasPrefix(p, "/console/v1/pod-monitor/") || strings.HasPrefix(p, "/console/v1/ingress-monitor/") || strings.HasPrefix(p, "/console/v1/flagger-monitor/")
	podLog := p == "/console/v1/pod/logs/get" || p == "/console/v1/pod/logs/download" || p == "/ws/v1/pod/logs/download" || p == "/ws/v1/pod/logs/stream" || p == "/console/v1/pod/containers" || p == "/console/v1/pod/default-container"
	logging := strings.HasPrefix(p, "/manager/v1/logging/")
	config := strings.HasPrefix(p, "/manager/v1/monitoring/")
	rules := strings.HasPrefix(p, "/workload/v1/monitoring/")
	if !monitor && !podLog && !logging && !config && !rules {
		return nil, false, nil
	}
	f, err := requestFields(r)
	if err != nil {
		return nil, true, err
	}
	c := requestCheck{target: Target{ClusterUUID: field(f, "clusterUuid"), Namespace: field(f, "namespace"), ProjectUUID: field(f, "projectUuid")}, resource: "monitoring", action: "view"}
	if podLog {
		c.resource = "logging"
		if strings.HasSuffix(p, "/download") {
			c.action = "export"
		}
		c.workspaceID, err = strconv.ParseUint(uintField(f, "workloadId"), 10, 64)
		if err != nil || c.workspaceID == 0 {
			return nil, true, ErrDenied
		}
		return []requestCheck{c}, true, nil
	}
	if config {
		// Path-bound cluster wins; reject any competing client cluster field.
		parts := strings.Split(strings.TrimPrefix(p, "/manager/v1/monitoring/"), "/")
		if len(parts) != 3 || parts[2] != "config" || (c.target.ClusterUUID != "" && c.target.ClusterUUID != parts[1]) {
			return nil, true, ErrDenied
		}
		c.target = Target{ClusterUUID: parts[1]}
		c.action, c.aggregate = "manage", true // Config may contain credentials.
	} else if rules {
		// Arbitrary rule PromQL and probe targets are cluster-level administration.
		c.target.Namespace, c.target.ProjectUUID = "", ""
		c.action, c.aggregate = "manage", true
	} else if monitor {
		// Values interpolated in PromQL must not escape a quoted matcher/range.
		for _, raw := range f {
			var s string
			if json.Unmarshal(raw, &s) == nil && strings.ContainsAny(s, "\"\\\r\n\x00") {
				return nil, true, ErrDenied
			}
		}
		if strings.HasPrefix(p, "/console/v1/namespace-monitor/") {
			if strings.HasSuffix(p, "/compare") {
				// Use the same parser as the eventual handler for []string form data.
				names, err := comparisonNamespaces(r)
				if err != nil || len(names) == 0 || len(names) > 50 {
					return nil, true, ErrDenied
				}
				checks := make([]requestCheck, 0, len(names))
				for _, ns := range names {
					n := c
					n.target.Namespace = ns
					if !namespace(ns) {
						return nil, true, ErrDenied
					}
					checks = append(checks, n)
				}
				return checks, true, nil
			}
			if !namespace(c.target.Namespace) {
				return nil, true, ErrDenied
			}
		} else if !strings.HasPrefix(p, "/console/v1/pod-monitor/") {
			// Legacy cluster/node/ingress/flagger queries may ignore Namespace.
			c.target.Namespace, c.target.ProjectUUID = "", ""
		}
		if strings.HasSuffix(p, "/container/environment") {
			c.action = "manage"
		}
		c.aggregate = c.target.Namespace == ""
	} else if logging {
		c.resource = "logging"
		suffix := strings.TrimPrefix(p, "/manager/v1/logging")
		switch suffix {
		case "/query", "/download", "/labels/keys", "/labels/values":
			if suffix == "/download" {
				c.action = "export"
			}
			// Expressions are not safely rewritable to a namespace. They require
			// platform-wide access, including when a misleading namespace is sent.
			raw := (field(f, "mode") != "" && field(f, "mode") != "form") || field(f, "expr") != "" || field(f, "command") != "" || ((suffix == "/query" || suffix == "/download") && field(f, "queryMode") != "" && field(f, "queryMode") != "platform")
			system := field(f, "sourceType") == "system" || field(f, "logType") == "system" || field(f, "logSourceType") == "system"
			if raw {
				c.global = true
				c.target = Target{}
			} else if system {
				c.target.Namespace, c.target.ProjectUUID = "", ""
			}
			// Workspace-only labels aren't scoped by the RPC. Require explicit ns.
			if rawID := uintField(f, "workspaceId"); rawID != "" && rawID != "0" {
				c.workspaceID, err = strconv.ParseUint(rawID, 10, 64)
				if err != nil || c.workspaceID == 0 || c.target.Namespace == "" {
					return nil, true, ErrDenied
				}
			}
		case "/query-mode/options": // Backend availability only, no query results.
		case "/cluster-config", "/cluster-sync-status":
			c.action = "manage"
			c.target.Namespace, c.target.ProjectUUID = "", ""
		default:
			// Alert/event IDs are not client-bound targets. Until the record
			// resolver is integrated, only unrestricted platform administrators.
			c.global = true
			c.target = Target{}
			c.action = "manage"
		}
		c.aggregate = c.target.Namespace == ""
	}
	return []requestCheck{c}, true, nil
}
