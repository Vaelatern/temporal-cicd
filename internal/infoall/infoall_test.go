package infoall

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestHandlerReportsTreeAndRoutes(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hi"), 0644); err != nil {
		t.Fatal(err)
	}
	h := Handler{
		Service: "cache",
		Routes:  []string{"GET /.vaelcicd/info/all"},
		Roots:   map[string]string{"repos": dir},
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest("GET", "/.vaelcicd/info/all", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d", rr.Code)
	}
	var rep Report
	if err := json.Unmarshal(rr.Body.Bytes(), &rep); err != nil {
		t.Fatal(err)
	}
	if rep.Service != "cache" || len(rep.Routes) != 1 {
		t.Fatalf("bad report: %+v", rep)
	}
	n, ok := rep.Trees["repos"]
	if !ok || n.Type != "dir" || len(n.Children) != 1 || n.Children[0].Name != "a.txt" {
		t.Fatalf("bad tree: %+v", n)
	}
}
