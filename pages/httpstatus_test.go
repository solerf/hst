package pages

import (
	"bytes"
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

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
		"Cloudflare":                        8,
		"AWS Elastic Load Balancing":        1,
		"Caching warning codes (obsoleted)": 7,
	}

	type assertion = func() string
	assertions := func(p *HttpStatusCodePage) []assertion {
		return []assertion{
			func() string {
				if p.Revision != 1317392915 {
					return fmt.Sprintf("unexpected [Revision] %v", p.Revision)
				}
				return ""
			},
			func() string {
				if len(expectations) != len(p.CodeTypes) {
					return fmt.Sprintf("invalid [CodeTypes] size: expected %v, got %v", len(expectations), len(p.CodeTypes))
				}
				return ""
			},
			func() string {
				var results []string
				for k, size := range expectations {
					idx := slices.IndexFunc(p.CodeTypes, func(codeType HttpStatusCodeType) bool {
						return codeType.Type == k
					})

					if idx == -1 {
						results = append(results, fmt.Sprintf("missing [%v] in page", k))
						continue
					}

					actualSize := len(p.CodeTypes[idx].Codes)
					if size != actualSize {
						results = append(results, fmt.Sprintf("invalid [%v] size: expected %v, got %v", k, size, actualSize))
						continue
					}
				}
				return strings.Join(results, "\n")
			},
			func() string {
				var results []string
				for _, ct := range p.CodeTypes {
					_, ok := expectations[ct.Type]
					if !ok {
						results = append(results, fmt.Sprintf("unexpected [%v] in page", ct.Type))
						continue
					}
				}
				return strings.Join(results, "\n")
			},
		}
	}

	assertionsRun := func(assertions []assertion) []string {
		failures := make([]string, 0, 10)
		for _, a := range assertions {
			if r := a(); len(r) > 0 {
				failures = append(failures, r)
			}
		}
		return failures
	}

	file, err := os.ReadFile("testdata/httpstatus_page.html")
	if err != nil {
		t.Fatal(err)
	}

	hDoc, err := html.Parse(bytes.NewReader(file))
	if err != nil {
		t.Fatal(err)
	}

	page := ParseHttpStatusCodesPage(hDoc)
	result := assertionsRun(assertions(page))
	if len(result) > 0 {
		t.Errorf("\n%v", strings.Join(result, "\n"))
	}
}

func Benchmark_Parse_HttpStatus_Page(b *testing.B) {
	file, err := os.ReadFile("testdata/httpstatus_page.html")
	if err != nil {
		b.Fatal(err)
	}

	hDoc, err := html.Parse(bytes.NewReader(file))
	if err != nil {
		b.Fatal(err)
	}

	for i := 0; i < b.N; i++ {
		ParseHttpStatusCodesPage(hDoc)
	}
}
