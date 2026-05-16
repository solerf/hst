package pages

import (
	"bytes"
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/html"
)

func loadFixture(tb testing.TB) *html.Node {
	tb.Helper()

	file, err := os.ReadFile("../testdata/httpstatus_page.html")
	if err != nil {
		tb.Fatal(err)
	}

	hDoc, err := html.Parse(bytes.NewReader(file))
	if err != nil {
		tb.Fatal(err)
	}
	return hDoc
}

func samplePage() *HttpStatusCodePage {
	return &HttpStatusCodePage{
		Revision: 42,
		Date:     time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		CodeTypes: []HttpStatusCodeType{
			{
				Type: "4xx client errors",
				Codes: []HttpStatusCode{
					{Code: "404", Name: "NotFound", Description: "missing"},
					{Code: "401", Name: "Unauthorized", Description: "auth"},
				},
			},
			{
				Type: "5xx server errors",
				Codes: []HttpStatusCode{
					{Code: "500", Name: "InternalServerError", Description: "boom"},
				},
			},
		},
	}
}

func Test_Parse_HttpStatus_Page(t *testing.T) {
	expectations := map[string]int{
		"1xx informational response":        4,
		"2xx success":                       10,
		"3xx redirection":                   9,
		"4xx client errors":                 29,
		"5xx server errors":                 11,
		"Nonstandard codes":                 22,
		"Internet Information Services":     3,
		"nginx":                             6,
		"Cloudflare":                        9,
		"AWS Elastic Load Balancing":        5,
		"Caching warning codes (obsoleted)": 7,
	}

	page := ParseHttpStatusCodesPage(loadFixture(t), 1317392915)

	if page.Revision != 1317392915 {
		t.Errorf("unexpected [Revision] `%v`", page.Revision)
	}

	if len(expectations) != len(page.CodeTypes) {
		t.Errorf("invalid [CodeTypes] size: expected `%v`, got `%v`", len(expectations), len(page.CodeTypes))
	}

	var failures []string
	for k, size := range expectations {
		idx := slices.IndexFunc(page.CodeTypes, func(codeType HttpStatusCodeType) bool {
			return codeType.Type == k
		})
		if idx == -1 {
			failures = append(failures, fmt.Sprintf("missing [`%v`] in page", k))
			continue
		}

		if got := len(page.CodeTypes[idx].Codes); got != size {
			failures = append(failures, fmt.Sprintf("invalid [`%v`] size: expected `%v`, got `%v`", k, size, got))
		}
	}
	for _, ct := range page.CodeTypes {
		if _, ok := expectations[ct.Type]; !ok {
			failures = append(failures, fmt.Sprintf("unexpected [`%v`] in page", ct.Type))
		}
	}

	if len(failures) > 0 {
		t.Errorf("\n%v", strings.Join(failures, "\n"))
	}
}

func Test_ByType(t *testing.T) {
	scenario := []struct {
		name     string
		filter   string
		wantType string
		wantHit  bool
	}{
		{"exact match", "4xx client errors", "4xx client errors", true},
		{"case insensitive", "5XX SERVER", "5xx server errors", true},
		{"substring", "client", "4xx client errors", true},
		{"miss leaves page unchanged", "totally-missing", "", false},
	}

	for _, s := range scenario {
		t.Run(s.name, func(t *testing.T) {
			p := samplePage()
			originalLen := len(p.CodeTypes)

			got, ok := p.ByStatusType(s.filter)

			if ok != s.wantHit {
				t.Fatalf("expected hit=%v, got %v", s.wantHit, ok)
			}

			if s.wantHit {
				if len(got.CodeTypes) != 1 {
					t.Fatalf("expected 1 CodeType, got %d", len(got.CodeTypes))
				}
				if got.CodeTypes[0].Type != s.wantType {
					t.Fatalf("expected type %q, got %q", s.wantType, got.CodeTypes[0].Type)
				}
			} else {
				if len(got.CodeTypes) != originalLen {
					t.Fatalf("expected no narrowing on miss, got %d (was %d)", len(got.CodeTypes), originalLen)
				}
			}
		})
	}
}

func Test_ByCode(t *testing.T) {
	scenario := []struct {
		name     string
		filter   string
		wantType string
		wantCode string
		wantHit  bool
	}{
		{"exact", "404", "4xx client errors", "404", true},
		{"case insensitive", "7xx", "", "", false}, // sample has no "5xx" string code
		{"miss leaves page unchanged", "999", "", "", false},
		{"5xx category", "500", "5xx server errors", "500", true},
	}

	for _, s := range scenario {
		t.Run(s.name, func(t *testing.T) {
			p := samplePage()

			got, ok := p.ByPrefixCode(s.filter)

			if ok != s.wantHit {
				t.Fatalf("expected hit=%v, got %v", s.wantHit, ok)
			}

			if s.wantHit {
				if len(got.CodeTypes) != 1 || len(got.CodeTypes[0].Codes) != 1 {
					t.Fatalf("expected 1 type with 1 code, got %d types", len(got.CodeTypes))
				}
				if got.CodeTypes[0].Type != s.wantType || got.CodeTypes[0].Codes[0].Code != s.wantCode {
					t.Fatalf("expected (%q, %q), got (%q, %q)",
						s.wantType, s.wantCode,
						got.CodeTypes[0].Type, got.CodeTypes[0].Codes[0].Code)
				}
			} else {
				if len(got.CodeTypes) != 0 {
					t.Fatalf("expected empty result on miss, got %d code types", len(got.CodeTypes))
				}
			}
		})
	}
}

func Test_ToJson_OnlyCodeTypes(t *testing.T) {
	p := samplePage()
	out, err := p.ToJson()
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(out, `"4xx client errors"`) {
		t.Errorf("expected 4xx category in output, got: %s", out)
	}
	if strings.Contains(out, `"revision"`) || strings.Contains(out, `"date"`) {
		t.Errorf("ToJson should expose only CodeTypes (no revision/date), got: %s", out)
	}
}

func Test_Serialize_Deserialize_Roundtrip(t *testing.T) {
	p := samplePage()

	encoded, err := p.Serialize()
	if err != nil {
		t.Fatal(err)
	}

	got, err := Deserialize([]byte(encoded))
	if err != nil {
		t.Fatal(err)
	}

	if got.Revision != p.Revision {
		t.Errorf("revision: expected %d, got %d", p.Revision, got.Revision)
	}
	if !got.Date.Equal(p.Date) {
		t.Errorf("date: expected %v, got %v", p.Date, got.Date)
	}
	if len(got.CodeTypes) != len(p.CodeTypes) {
		t.Fatalf("code types: expected %d, got %d", len(p.CodeTypes), len(got.CodeTypes))
	}
	for i := range p.CodeTypes {
		if got.CodeTypes[i].Type != p.CodeTypes[i].Type {
			t.Errorf("type[%d]: expected %q, got %q", i, p.CodeTypes[i].Type, got.CodeTypes[i].Type)
		}
		if len(got.CodeTypes[i].Codes) != len(p.CodeTypes[i].Codes) {
			t.Errorf("codes[%d]: expected %d, got %d", i, len(p.CodeTypes[i].Codes), len(got.CodeTypes[i].Codes))
		}
	}
}

func Test_Deserialize_Errors(t *testing.T) {
	scenario := []struct {
		name             string
		in               []byte
		expectedContains string
	}{
		{"empty bytes", []byte(""), "empty bytes"},
		{"invalid base64", []byte("not-base64-!@#$"), "decoding hst"},
		{"valid base64 but not JSON", []byte("bm90LWpzb24="), "unmarshal hst"},
	}

	for _, s := range scenario {
		t.Run(s.name, func(t *testing.T) {
			_, err := Deserialize(s.in)
			if err == nil {
				t.Fatal("expected error, got none")
			}
			if !strings.Contains(err.Error(), s.expectedContains) {
				t.Fatalf("expected error containing %q, got: %v", s.expectedContains, err)
			}
		})
	}
}

func Benchmark_Parse_HttpStatus_Page(b *testing.B) {
	hDoc := loadFixture(b)

	for b.Loop() {
		ParseHttpStatusCodesPage(hDoc, 0)
	}
}
