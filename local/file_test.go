package local

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/solerf/hst/pages"
)

func samplePage() *pages.HttpStatusCodePage {
	return &pages.HttpStatusCodePage{
		Revision: 7,
		Date:     time.Date(2026, 5, 16, 12, 0, 0, 0, time.UTC),
		CodeTypes: []pages.HttpStatusCodeType{
			{
				Type: "4xx client errors",
				Codes: []pages.HttpStatusCode{
					{Code: "404", Name: "NotFound", Description: "missing"},
				},
			},
		},
	}
}

func Test_Write_Then_SourceHTML_Roundtrip(t *testing.T) {
	dir := t.TempDir()
	p := samplePage()

	if err := Write(dir, p); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	info, err := os.Stat(filepath.Join(dir, ".hst"))
	if err != nil {
		t.Fatalf("expected .hst to exist: %v", err)
	}
	if mode := info.Mode().Perm(); mode != 0600 {
		t.Errorf("expected file mode 0600, got %o", mode)
	}

	got, err := SourceHTML(dir)
	if err != nil {
		t.Fatalf("SourceHTML failed: %v", err)
	}

	if got.Revision != p.Revision {
		t.Errorf("revision: expected %d, got %d", p.Revision, got.Revision)
	}
	if !got.Date.Equal(p.Date) {
		t.Errorf("date: expected %v, got %v", p.Date, got.Date)
	}
	if len(got.CodeTypes) != 1 || len(got.CodeTypes[0].Codes) != 1 {
		t.Fatalf("unexpected shape: %+v", got)
	}
	if got.CodeTypes[0].Codes[0].Code != "404" {
		t.Errorf("expected 404, got %q", got.CodeTypes[0].Codes[0].Code)
	}
}

func Test_SourceHTML_Missing_Wraps_ErrNotExist(t *testing.T) {
	dir := t.TempDir()

	_, err := SourceHTML(dir)
	if err == nil {
		t.Fatal("expected error for missing cache file")
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("expected wrapped os.ErrNotExist, got: %v", err)
	}
}

func Test_SourceHTML_Corrupt_File(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".hst"), []byte("not-valid-base64-!@#"), 0600); err != nil {
		t.Fatal(err)
	}

	if _, err := SourceHTML(dir); err == nil {
		t.Fatal("expected decoding error for corrupt cache")
	}
}

func Test_Write_Overwrites(t *testing.T) {
	dir := t.TempDir()

	first := samplePage()
	if err := Write(dir, first); err != nil {
		t.Fatal(err)
	}

	second := samplePage()
	second.Revision = 99
	if err := Write(dir, second); err != nil {
		t.Fatal(err)
	}

	got, err := SourceHTML(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got.Revision != 99 {
		t.Errorf("expected revision 99 after overwrite, got %d", got.Revision)
	}
}
