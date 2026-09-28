package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"go.temporal.io/sdk/client"
	temporal_envconfig "go.temporal.io/sdk/contrib/envconfig"
)

type Endpoint struct {
	Service string
	Method  string
	Path    string
	Body    string // example body
	Notes   string
}

type ServiceCfg struct {
	Name string
	URL  string
}

type UI struct {
	services  []ServiceCfg
	endpoints []Endpoint
	token     string // default token used for /info/all fetches
	baseURL   string // UI path prefix, e.g. /ui (no trailing slash)
	temporal  client.Client
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

// trimBaseURL mirrors awxy: "/" means no prefix; otherwise strip trailing slash.
func trimBaseURL(baseURL string) string {
	if baseURL == "" || baseURL == "/" {
		return ""
	}
	if !strings.HasPrefix(baseURL, "/") {
		baseURL = "/" + baseURL
	}
	return strings.TrimRight(baseURL, "/")
}

func withBaseURL(base string, h http.Handler) http.Handler {
	if base == "" {
		return h
	}
	stripped := http.StripPrefix(base, h)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == base {
			http.Redirect(w, r, base+"/", http.StatusFound)
			return
		}
		stripped.ServeHTTP(w, r)
	})
}

func main() {
	ui := &UI{
		baseURL: trimBaseURL(env("BASE_URL", "")),
		services: []ServiceCfg{
			{Name: "cache", URL: strings.TrimRight(env("TCD_CACHE_URL", "http://localhost:8080"), "/")},
			{Name: "kickoff", URL: strings.TrimRight(env("TCD_KICKOFF_URL", "http://localhost:8081"), "/")},
			{Name: "artifacts", URL: strings.TrimRight(env("TCD_ARTIFACTS_URL", "http://localhost:8082"), "/")},
		},
		token: env("TCD_SLOP_TOKEN", ""),
		endpoints: []Endpoint{
			{Service: "cache", Method: "GET", Path: "/.vaelcicd/info/all", Notes: "admin dump"},
			{Service: "cache", Method: "PUT", Path: "/sync/{repo}", Body: `{"url":"git@github.com:owner/repo.git","ssh-reading-private-key":""}`, Notes: "register repo"},
			{Service: "cache", Method: "POST", Path: "/sync/{repo}", Body: `{"url":"git@github.com:owner/repo.git","ssh-reading-private-key":""}`, Notes: "adjust repo"},
			{Service: "cache", Method: "POST", Path: "/sync/{repo}/{ref}", Notes: "fetch ref"},
			{Service: "cache", Method: "GET", Path: "/download/{repo}/{ref}", Notes: "tarball"},
			{Service: "artifacts", Method: "GET", Path: "/.vaelcicd/info/all", Notes: "admin dump"},
			{Service: "artifacts", Method: "PUT", Path: "/{path...}", Notes: "upload artifact"},
			{Service: "artifacts", Method: "GET", Path: "/{path...}", Notes: "download artifact"},
			{Service: "kickoff", Method: "GET", Path: "/.vaelcicd/info/all", Notes: "admin dump"},
			{Service: "kickoff", Method: "KICKOFF", Path: "/{repo}/{ref}", Body: `{"repository":"","ref":"","build-pattern":"MakeBuildUpload","compat-patch":""}`, Notes: "start build"},
			{Service: "kickoff", Method: "KICKOFF", Path: "/", Body: `{"repository":"repo","ref":"main","build-pattern":"MakeBuildUpload","compat-patch":""}`, Notes: "start build (body-driven)"},
			{Service: "kickoff", Method: "POST", Path: "/hooks/{source}/{repo}", Body: `{}`, Notes: "provider webhook"},
		},
	}

	opts := temporal_envconfig.MustLoadDefaultClientOptions()
	if c, err := client.Dial(opts); err != nil {
		log.Printf("[slop-ui] temporal unavailable: %v (builder/deployer panels will error)", err)
	} else {
		ui.temporal = c
		defer c.Close()
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", ui.home)
	mux.HandleFunc("GET /api/call", ui.callForm) // htmx fragment
	mux.HandleFunc("POST /api/call", ui.doCall)
	mux.HandleFunc("GET /api/curl", ui.curlPreview)
	mux.HandleFunc("GET /api/info/{service}", ui.proxyInfo)
	mux.HandleFunc("GET /api/temporal", ui.temporalStatus)

	listen := env("TCD_LISTEN", ":8090")
	log.Printf("[slop-ui] Listening on %s base=%q (admin UI, no auth)", listen, ui.baseURL)
	log.Fatal(http.ListenAndServe(listen, withBaseURL(ui.baseURL, mux)))
}

func (ui *UI) svcURL(name string) string {
	for _, s := range ui.services {
		if s.Name == name {
			return s.URL
		}
	}
	return ""
}

func (ui *UI) home(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		token = ui.token
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = pageTmpl.Execute(w, map[string]any{
		"Services":  ui.services,
		"Endpoints": ui.endpoints,
		"Token":     token,
	})
}

func (ui *UI) proxyInfo(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("service")
	base := ui.svcURL(name)
	if base == "" {
		http.Error(w, "unknown service", 404)
		return
	}
	token := r.URL.Query().Get("token")
	if token == "" {
		token = ui.token
	}
	req, _ := http.NewRequestWithContext(r.Context(), "GET", base+"/.vaelcicd/info/all", nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, err.Error(), 502)
		return
	}
	defer resp.Body.Close()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func (ui *UI) temporalStatus(w http.ResponseWriter, r *http.Request) {
	type qstat struct {
		TaskQueue string   `json:"task_queue"`
		Kind      string   `json:"kind"`
		Pollers   int      `json:"pollers"`
		Backlog   int64    `json:"backlog"`
		Error     string   `json:"error,omitempty"`
		PollerIds []string `json:"poller_identities,omitempty"`
	}
	out := map[string]any{"queues": []qstat{}}
	if ui.temporal == nil {
		out["error"] = "no temporal client"
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	queues := []struct {
		name, kind string
		tqType     client.TaskQueueType
	}{
		{"basic-builder", "workflow", client.TaskQueueTypeWorkflow},
		{"basic-builder", "activity", client.TaskQueueTypeActivity},
		{"deployer", "workflow", client.TaskQueueTypeWorkflow},
		{"deployer", "activity", client.TaskQueueTypeActivity},
	}
	var stats []qstat
	for _, q := range queues {
		s := qstat{TaskQueue: q.name, Kind: q.kind}
		desc, err := ui.temporal.DescribeTaskQueueEnhanced(ctx, client.DescribeTaskQueueEnhancedOptions{
			TaskQueue:      q.name,
			TaskQueueTypes: []client.TaskQueueType{q.tqType},
			ReportPollers:  true,
			ReportStats:    true,
			Versions:       &client.TaskQueueVersionSelection{Unversioned: true, AllActive: true},
		})
		if err != nil {
			s.Error = err.Error()
			stats = append(stats, s)
			continue
		}
		for _, vinfo := range desc.VersionsInfo {
			if ti, ok := vinfo.TypesInfo[q.tqType]; ok {
				s.Pollers += len(ti.Pollers)
				for _, p := range ti.Pollers {
					s.PollerIds = append(s.PollerIds, p.Identity)
				}
				if ti.Stats != nil {
					s.Backlog += ti.Stats.ApproximateBacklogCount
				}
			}
		}
		stats = append(stats, s)
	}
	out["queues"] = stats
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

func (ui *UI) callForm(w http.ResponseWriter, r *http.Request) {
	svc := r.URL.Query().Get("service")
	method := r.URL.Query().Get("method")
	path := r.URL.Query().Get("path")
	body := ""
	for _, e := range ui.endpoints {
		if e.Service == svc && e.Method == method && e.Path == path {
			body = e.Body
			break
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = formTmpl.Execute(w, map[string]any{
		"Service": svc,
		"Method":  method,
		"Path":    path,
		"Body":    body,
		"Base":    ui.svcURL(svc),
		"Token":   r.URL.Query().Get("token"),
	})
}

func buildCurl(method, fullURL, token, body string) string {
	var b strings.Builder
	b.WriteString("curl -sS -X ")
	b.WriteString(shellQuote(method))
	if token != "" {
		b.WriteString(" -H ")
		b.WriteString(shellQuote("Authorization: Bearer " + token))
	}
	if body != "" {
		b.WriteString(" -H 'Content-Type: application/json'")
		b.WriteString(" --data-binary ")
		b.WriteString(shellQuote(body))
	}
	b.WriteString(" ")
	b.WriteString(shellQuote(fullURL))
	return b.String()
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func (ui *UI) curlPreview(w http.ResponseWriter, r *http.Request) {
	svc := r.FormValue("service")
	method := r.FormValue("method")
	path := r.FormValue("path")
	token := r.FormValue("token")
	body := r.FormValue("body")
	base := ui.svcURL(svc)
	full := base + path
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprint(w, buildCurl(method, full, token, body))
}

func (ui *UI) doCall(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	svc := r.FormValue("service")
	method := r.FormValue("method")
	path := r.FormValue("path")
	token := r.FormValue("token")
	body := r.FormValue("body")
	base := ui.svcURL(svc)
	if base == "" {
		http.Error(w, "unknown service", 400)
		return
	}
	full := base + path
	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}
	req, err := http.NewRequestWithContext(r.Context(), method, full, bodyReader)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, `<pre class="err">request error: %s</pre><pre class="curl">%s</pre>`,
			html.EscapeString(err.Error()), html.EscapeString(buildCurl(method, full, token, body)))
		return
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	ct := resp.Header.Get("Content-Type")
	pretty := string(raw)
	if strings.Contains(ct, "json") || json.Valid(raw) {
		var buf bytes.Buffer
		if err := json.Indent(&buf, raw, "", "  "); err == nil {
			pretty = buf.String()
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<div class="resp"><div class="status">HTTP %d %s</div><pre class="curl">%s</pre><pre>%s</pre></div>`,
		resp.StatusCode, html.EscapeString(resp.Status),
		html.EscapeString(buildCurl(method, full, token, body)),
		html.EscapeString(pretty))
}

var pageTmpl = template.Must(template.New("page").Parse(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>slop-ui · lrcicd</title>
<script src="https://unpkg.com/htmx.org@2.0.4"></script>
<style>
  :root { color-scheme: dark light; --bg:#0f1115; --card:#1a1f29; --fg:#e6e6e6; --muted:#9aa3b2; --acc:#6ee7b7; --err:#f87171; --bd:#2a3140; }
  * { box-sizing: border-box; }
  body { margin:0; font: 14px/1.45 ui-sans-serif, system-ui, sans-serif; background:var(--bg); color:var(--fg); }
  header { padding:1rem 1.25rem; border-bottom:1px solid var(--bd); display:flex; gap:1rem; align-items:baseline; flex-wrap:wrap; }
  header h1 { margin:0; font-size:1.1rem; letter-spacing:.02em; }
  header .muted { color:var(--muted); }
  main { display:grid; grid-template-columns: 280px 1fr; gap:0; min-height: calc(100vh - 56px); }
  nav { border-right:1px solid var(--bd); padding:1rem; overflow:auto; }
  section { padding:1rem 1.25rem; overflow:auto; }
  .svc { margin-bottom:1.25rem; }
  .svc h2 { margin:0 0 .4rem; font-size:.85rem; text-transform:uppercase; letter-spacing:.08em; color:var(--muted); }
  button.ep, a.ep { display:block; width:100%; text-align:left; background:transparent; border:1px solid transparent; color:var(--fg); padding:.35rem .5rem; border-radius:6px; cursor:pointer; font:inherit; }
  button.ep:hover, a.ep:hover { background:var(--card); border-color:var(--bd); }
  button.ep code { color:var(--acc); }
  .card { background:var(--card); border:1px solid var(--bd); border-radius:10px; padding:1rem; margin-bottom:1rem; }
  label { display:block; font-size:.75rem; color:var(--muted); margin:.5rem 0 .2rem; }
  input, textarea, select { width:100%; background:#0c0e13; color:var(--fg); border:1px solid var(--bd); border-radius:6px; padding:.45rem .55rem; font:inherit; }
  textarea { min-height:110px; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; }
  .row { display:flex; gap:.5rem; flex-wrap:wrap; align-items:end; }
  .row > * { flex:1; min-width:140px; }
  .actions { margin-top:.75rem; display:flex; gap:.5rem; flex-wrap:wrap; }
  .actions button { background:var(--acc); color:#042f1a; border:0; border-radius:6px; padding:.45rem .9rem; font-weight:600; cursor:pointer; }
  .actions button.secondary { background:transparent; color:var(--fg); border:1px solid var(--bd); }
  pre { background:#0c0e13; border:1px solid var(--bd); border-radius:8px; padding:.75rem; overflow:auto; white-space:pre-wrap; word-break:break-word; }
  pre.curl { color:var(--acc); }
  pre.err { color:var(--err); }
  .status { font-weight:600; margin-bottom:.5rem; }
  .tree { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size:12px; }
  .tree details { margin-left:.75rem; }
  .pill { display:inline-block; padding:.1rem .45rem; border-radius:999px; background:#0c0e13; border:1px solid var(--bd); font-size:.7rem; color:var(--muted); }
  #panels { display:grid; gap:1rem; }
  @media (max-width: 900px) { main { grid-template-columns: 1fr; } nav { border-right:0; border-bottom:1px solid var(--bd); } }
</style>
</head>
<body>
<header>
  <h1>slop-ui</h1>
  <span class="muted">admin visibility · every call takes its own token · curl always shown</span>
  <form style="margin-left:auto;display:flex;gap:.4rem;align-items:center" onsubmit="location.search='?token='+encodeURIComponent(this.token.value);return false">
    <label for="tok" style="margin:0">default token</label>
    <input id="tok" name="token" value="{{.Token}}" style="width:220px" placeholder="Bearer for /info/all">
    <button class="secondary" style="background:transparent;color:var(--fg);border:1px solid var(--bd);border-radius:6px;padding:.35rem .7rem;cursor:pointer">set</button>
  </form>
</header>
<main>
<nav>
  {{range .Endpoints}}
  <button class="ep"
    hx-get="api/call?service={{.Service}}&method={{urlquery .Method}}&path={{urlquery .Path}}&token={{urlquery $.Token}}"
    hx-target="#work"
    hx-swap="innerHTML">
    <span class="pill">{{.Service}}</span> <code>{{.Method}}</code> {{.Path}}
    {{if .Notes}}<div class="muted" style="font-size:.7rem">{{.Notes}}</div>{{end}}
  </button>
  {{end}}
</nav>
<section>
  <div id="panels">
    <div class="card">
      <h3 style="margin-top:0">service /.vaelcicd/info/all</h3>
      <div class="row">
        {{range .Services}}
        <button class="secondary" style="background:transparent;color:var(--fg);border:1px solid var(--bd);border-radius:6px;padding:.45rem .7rem;cursor:pointer"
          hx-get="api/info/{{.Name}}?token={{urlquery $.Token}}"
          hx-target="#info-out"
          hx-swap="innerHTML"
          hx-on::after-request="(function(t){try{t.textContent=JSON.stringify(JSON.parse(event.detail.xhr.responseText),null,2)}catch(e){t.textContent=event.detail.xhr.responseText}})(document.getElementById('info-out'))">
          refresh {{.Name}}
        </button>
        {{end}}
      </div>
      <pre id="info-out" class="tree muted">click a service to load /.vaelcicd/info/all</pre>
    </div>
    <div class="card">
      <h3 style="margin-top:0">temporal queues (builder / deployer)</h3>
      <button class="secondary" style="background:transparent;color:var(--fg);border:1px solid var(--bd);border-radius:6px;padding:.45rem .7rem;cursor:pointer;margin-bottom:.5rem"
        hx-get="api/temporal"
        hx-target="#tq-out"
        hx-swap="innerHTML"
        hx-on::after-request="(function(t){try{t.textContent=JSON.stringify(JSON.parse(event.detail.xhr.responseText),null,2)}catch(e){t.textContent=event.detail.xhr.responseText}})(document.getElementById('tq-out'))">
        refresh queues
      </button>
      <pre id="tq-out" class="muted">pollers ≈ workers; backlog ≈ outstanding tasks</pre>
    </div>
  </div>
  <div class="card" id="work">
    <p class="muted" style="margin:0">pick an endpoint on the left</p>
  </div>
</section>
</main>
</body>
</html>`))

var formTmpl = template.Must(template.New("form").Parse(`
<h3 style="margin-top:0"><span class="pill">{{.Service}}</span> <code>{{.Method}}</code> {{.Path}}</h3>
<p class="muted" style="margin-top:0">base: {{.Base}}</p>
<form hx-post="api/call" hx-target="#result" hx-swap="innerHTML"
      hx-on:input="htmx.ajax('GET','api/curl?'+new URLSearchParams(new FormData(this)).toString(),{target:'#curl',swap:'innerHTML'})"
      hx-on:change="htmx.ajax('GET','api/curl?'+new URLSearchParams(new FormData(this)).toString(),{target:'#curl',swap:'innerHTML'})">
  <input type="hidden" name="service" value="{{.Service}}">
  <input type="hidden" name="method" value="{{.Method}}">
  <div class="row">
    <div>
      <label>path (edit placeholders)</label>
      <input name="path" value="{{.Path}}" required>
    </div>
    <div>
      <label>token for this call</label>
      <input name="token" value="{{.Token}}" placeholder="Bearer token" autocomplete="off">
    </div>
  </div>
  <label>body</label>
  <textarea name="body">{{.Body}}</textarea>
  <div class="actions">
    <button type="submit">send</button>
    <button type="button" class="secondary"
      onclick="htmx.ajax('GET','api/curl?'+new URLSearchParams(new FormData(this.form)).toString(),{target:'#curl',swap:'innerHTML'})">
      refresh curl
    </button>
  </div>
</form>
<label>curl equivalent</label>
<pre id="curl" class="curl">{{/* filled on first paint via hx */}}</pre>
<div id="result"></div>
<script>
(function(){
  const f = document.querySelector('#work form');
  if (f) htmx.ajax('GET','api/curl?'+new URLSearchParams(new FormData(f)).toString(),{target:'#curl',swap:'innerHTML'});
})();
</script>
`))
