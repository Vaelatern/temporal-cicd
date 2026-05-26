package main

import (
	"testing"
)

func TestExtractTypeFromDisposition(t *testing.T) {
	tests := []struct {
		name        string
		disposition string
		want        string
	}{
		{
			name:        "tgz with quotes",
			disposition: `attachment; filename="myrepo-main.tgz"`,
			want:        "tgz",
		},
		{
			name:        "tgz without quotes",
			disposition: "attachment; filename=myrepo-main.tgz",
			want:        "tgz",
		},
		{
			name:        "tgz with spaces in repo and ref",
			disposition: `attachment; filename="my repo-feature branch.tgz"`,
			want:        "tgz",
		},
		{
			name:        "tar.gz extension",
			disposition: `attachment; filename="archive.tar.gz"`,
			want:        "gz",
		},
		{
			name:        "zip extension",
			disposition: `attachment; filename="archive.zip"`,
			want:        "zip",
		},
		{
			name:        "empty disposition",
			disposition: "",
			want:        "",
		},
		{
			name:        "no filename parameter",
			disposition: "attachment",
			want:        "",
		},
		{
			name:        "filename with no extension",
			disposition: `attachment; filename="Makefile"`,
			want:        "",
		},
		{
			name:        "filename with dotfile",
			disposition: `attachment; filename=".gitignore"`,
			want:        "gitignore",
		},
		{
			name:        "uppercase extension",
			disposition: `attachment; filename="archive.TGZ"`,
			want:        "tgz",
		},
		{
			name:        "invalid disposition format",
			disposition: "not-a-valid-disposition",
			want:        "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractTypeFromDisposition(tt.disposition)
			if got != tt.want {
				t.Errorf("extractTypeFromDisposition(%q) = %q, want %q", tt.disposition, got, tt.want)
			}
		})
	}
}

func TestExtractTypeFromDispositionWithFilenameStar(t *testing.T) {
	disposition := `attachment; filename="myrepo-main.tgz"; filename*=UTF-8''myrepo-main.tgz`
	got := extractTypeFromDisposition(disposition)
	want := "tgz"
	if got != want {
		t.Errorf("extractTypeFromDisposition(%q) = %q, want %q", disposition, got, want)
	}
}
