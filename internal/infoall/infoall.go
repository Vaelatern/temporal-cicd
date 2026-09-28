package infoall

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
)

// ponytail: caps keep responses small; raise if ops need deeper dumps.
const maxDepth = 6
const maxEntries = 500

type Node struct {
	Name     string `json:"name"`
	Type     string `json:"type"` // file|dir
	Size     int64  `json:"size,omitempty"`
	Children []Node `json:"children,omitempty"`
	Truncated bool  `json:"truncated,omitempty"`
}

type Report struct {
	Service string            `json:"service"`
	Routes  []string          `json:"routes"`
	Trees   map[string]Node   `json:"trees"`
	Meta    map[string]string `json:"meta,omitempty"`
}

type Handler struct {
	Service string
	Routes  []string
	Roots   map[string]string // label -> absolute/relative path
	Meta    map[string]string
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	trees := make(map[string]Node, len(h.Roots))
	entriesLeft := maxEntries
	for label, root := range h.Roots {
		trees[label] = walk(root, filepath.Base(root), 0, &entriesLeft)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(Report{
		Service: h.Service,
		Routes:  h.Routes,
		Trees:   trees,
		Meta:    h.Meta,
	})
}

func walk(path, name string, depth int, left *int) Node {
	if *left <= 0 {
		return Node{Name: name, Type: "dir", Truncated: true}
	}
	*left--

	fi, err := os.Lstat(path)
	if err != nil {
		return Node{Name: name, Type: "missing"}
	}
	if !fi.IsDir() {
		return Node{Name: name, Type: "file", Size: fi.Size()}
	}
	n := Node{Name: name, Type: "dir"}
	if depth >= maxDepth {
		n.Truncated = true
		return n
	}
	ents, err := os.ReadDir(path)
	if err != nil {
		n.Truncated = true
		return n
	}
	for _, e := range ents {
		if *left <= 0 {
			n.Truncated = true
			break
		}
		child := walk(filepath.Join(path, e.Name()), e.Name(), depth+1, left)
		// skip permission noise from special types
		if e.Type()&fs.ModeSymlink != 0 && child.Type == "missing" {
			child.Type = "symlink"
		}
		n.Children = append(n.Children, child)
	}
	return n
}
