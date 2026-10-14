package config

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestParseDebugLevel(t *testing.T) {
	cases := map[string]DebugLevel{
		"":               DebugLevelNone,
		"false":          DebugLevelNone,
		"False":          DebugLevelNone,
		"0":              DebugLevelNone,
		"off":            DebugLevelNone,
		"true":           DebugLevelDebug,
		"True":           DebugLevelDebug,
		"debug":          DebugLevelDebug,
		"yes":            DebugLevelDebug,
		"show-sensitive": DebugLevelShowSensitive,
		"SHOW-SENSITIVE": DebugLevelShowSensitive,
	}
	for in, want := range cases {
		if got := parseDebugLevel(in); got != want {
			t.Fatalf("parseDebugLevel(%q)=%q want %q", in, got, want)
		}
	}
}

func TestDebugLevelUnmarshalYAML(t *testing.T) {
	cases := []struct {
		in   string
		want DebugLevel
	}{
		{"debug: false\n", DebugLevelNone},
		{"debug: True\n", DebugLevelDebug},
		{"debug: debug\n", DebugLevelDebug},
		{"debug: show-sensitive\n", DebugLevelShowSensitive},
		{"debug: null\n", DebugLevelNone},
		{"{}\n", DebugLevelNone},
	}
	for _, c := range cases {
		var conf Config
		if err := yaml.Unmarshal([]byte(c.in), &conf); err != nil {
			t.Fatalf("unmarshal %q: %v", c.in, err)
		}
		if conf.Debug != c.want {
			t.Fatalf("%q -> %q want %q", c.in, conf.Debug, c.want)
		}
	}
}

func TestDumpRedactsUnlessShowSensitive(t *testing.T) {
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	defer func() { os.Stdout = old }()

	conf := Config{
		Debug:  DebugLevelDebug,
		Listen: ":8080",
		Cache: CacheConfig{
			URL:     "http://localhost:8080",
			Headers: map[string]string{"Authorization": "Bearer secret"},
		},
	}
	conf.dumpWorkingConfig()
	w.Close()
	b, _ := io.ReadAll(r)
	os.Stdout = old
	s := string(b)
	if strings.Contains(s, "Bearer secret") {
		t.Fatalf("secret leaked: %s", s)
	}
	if !strings.Contains(s, "***") || !strings.Contains(s, "boot_time:") {
		t.Fatalf("expected redaction + boot_time: %s", s)
	}
	if conf.Cache.Headers["Authorization"] != "Bearer secret" {
		t.Fatal("dump mutated live config")
	}

	r, w, err = os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	conf.Debug = DebugLevelShowSensitive
	conf.dumpWorkingConfig()
	w.Close()
	b, _ = io.ReadAll(r)
	if !strings.Contains(string(b), "Bearer secret") {
		t.Fatalf("show-sensitive should keep token: %s", b)
	}
}

func TestDebugMiddleware(t *testing.T) {
	old := log.Writer()
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(old)

	hit := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hit = true })

	none := (&Config{Debug: DebugLevelNone}).DebugMiddleware(next)
	none.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/x", nil))
	if !hit || buf.Len() != 0 {
		t.Fatalf("none: hit=%v log=%q", hit, buf.String())
	}

	buf.Reset()
	hit = false
	debug := (&Config{Debug: DebugLevelDebug}).DebugMiddleware(next)
	req := httptest.NewRequest("GET", "/x?token=sekrit", nil)
	req.Header.Set("Authorization", "Bearer sekrit")
	req.RemoteAddr = "1.2.3.4:5"
	debug.ServeHTTP(httptest.NewRecorder(), req)
	s := buf.String()
	if !hit || !strings.Contains(s, "GET /x from 1.2.3.4:5") || strings.Contains(s, "sekrit") {
		t.Fatalf("debug: %q", s)
	}

	buf.Reset()
	hit = false
	sens := (&Config{Debug: DebugLevelShowSensitive}).DebugMiddleware(next)
	sens.ServeHTTP(httptest.NewRecorder(), req)
	s = buf.String()
	if !hit || !strings.Contains(s, `token="sekrit"`) {
		t.Fatalf("show-sensitive: %q", s)
	}
}
