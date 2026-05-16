package pages

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"slices"
	"strings"
	"time"
	"unicode"

	"golang.org/x/net/html"
)

type HttpStatusCode struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type HttpStatusCodeType struct {
	Type  string           `json:"type"`
	Codes []HttpStatusCode `json:"codes"`
}

type HttpStatusCodePage struct {
	Revision  int                  `json:"revision"`
	CodeTypes []HttpStatusCodeType `json:"code_types"`
	Date      time.Time            `json:"date"`
}

func (h *HttpStatusCodePage) ByStatusType(t string) (*HttpStatusCodePage, bool) {
	target := strings.ToLower(t)
	idx := slices.IndexFunc(h.CodeTypes, func(ct HttpStatusCodeType) bool {
		return strings.Contains(strings.ToLower(ct.Type), target)
	})

	if idx == -1 {
		return h, false
	}
	h.CodeTypes = h.CodeTypes[idx : idx+1]
	return h, true
}

func (h *HttpStatusCodePage) ByPrefixCode(c string) (*HttpStatusCodePage, bool) {
	var sb strings.Builder
	for _, r := range c {
		if unicode.IsDigit(r) {
			sb.WriteRune(r)
		}
	}

	if sb.Len() == 0 {
		return h, false
	}

	target := sb.String()
	for codeTypeIdx := range h.CodeTypes {
		codes := slices.DeleteFunc(h.CodeTypes[codeTypeIdx].Codes, func(code HttpStatusCode) bool {
			return !strings.HasPrefix(code.Code, target)
		})
		h.CodeTypes[codeTypeIdx].Codes = codes
	}

	h.CodeTypes = slices.DeleteFunc(h.CodeTypes, func(codeType HttpStatusCodeType) bool {
		return len(codeType.Codes) == 0
	})

	return h, len(h.CodeTypes) > 0
}

func (h *HttpStatusCodePage) ToJson() (string, error) {
	j, err := json.MarshalIndent(h.CodeTypes, " ", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal json: %w", err)
	}
	return string(j), nil
}

func (h *HttpStatusCodePage) Serialize() (string, error) {
	in, err := json.Marshal(h)
	if err != nil {
		return "", fmt.Errorf("marshal json hst: %w", err)
	}
	return base64.StdEncoding.EncodeToString(in), nil
}

func Deserialize(in []byte) (*HttpStatusCodePage, error) {
	if len(in) != 0 {
		out, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(in)))
		if err != nil {
			return nil, fmt.Errorf("decoding hst: %w", err)
		}

		h := &HttpStatusCodePage{}
		if err = json.Unmarshal(out, h); err != nil {
			return nil, fmt.Errorf("unmarshal hst: %w", err)
		}
		return h, nil
	}
	return nil, errors.New("empty bytes to deserialize")
}

func ParseHttpStatusCodesPage(htmlNode *html.Node, revision int) *HttpStatusCodePage {
	byType := make(map[string][]HttpStatusCode)

	for node := range iterHtmlNodes(htmlNode) {
		if node.Type == html.ElementNode && node.Data == "dl" {
			t := collectHttpStatusType(node)
			byType[t] = append(byType[t], collectHttpStatusCodes(node.ChildNodes())...)
		}
	}

	codeTypes := make([]HttpStatusCodeType, 0, len(byType))
	for t, codes := range byType {
		slices.SortFunc(codes, func(a, b HttpStatusCode) int {
			return strings.Compare(a.Code, b.Code)
		})
		codeTypes = append(codeTypes, HttpStatusCodeType{Type: t, Codes: codes})
	}

	slices.SortFunc(codeTypes, func(a, b HttpStatusCodeType) int {
		return strings.Compare(a.Type, b.Type)
	})

	return &HttpStatusCodePage{
		Revision:  revision,
		CodeTypes: codeTypes,
		Date:      time.Now(),
	}
}

func iterHtmlNodes(htmlNode *html.Node) iter.Seq[*html.Node] {
	return func(yield func(*html.Node) bool) {
		var walk func(*html.Node) bool
		walk = func(node *html.Node) bool {
			if node.Type == html.ElementNode && !yield(node) {
				return false
			}
			for c := node.FirstChild; c != nil; c = c.NextSibling {
				if !walk(c) {
					return false
				}
			}
			return true
		}
		walk(htmlNode)
	}
}

func collectHttpStatusType(node *html.Node) string {
	for sib := node.PrevSibling; sib != nil; sib = sib.PrevSibling {
		if sib.Type != html.ElementNode {
			continue
		}

		switch sib.Data {
		case "h1", "h2", "h3", "h4", "h5", "h6":
			return collectText(sib)
		}
	}

	if node.Parent != nil && node.Parent.FirstChild != nil {
		return collectText(node.Parent.FirstChild)
	}
	return ""
}

func collectHttpStatusCodes(nodes iter.Seq[*html.Node]) []HttpStatusCode {
	var codes []HttpStatusCode
	var code, name, description string

	for node := range nodes {
		if node.Type != html.ElementNode {
			continue
		}

		switch node.Data {
		case "dt":
			code, name, _ = strings.Cut(collectText(node), " ")
			code = strings.TrimSpace(code)
			name = strings.ReplaceAll(name, " ", "")

			if code == "" {
				code = "unknown"
			}
			if name == "" {
				name = "unknown"
			}

		case "dd":
			description = collectText(node)
		}

		if len(code) > 0 && len(name) > 0 && len(description) > 0 {
			codes = append(codes, HttpStatusCode{
				Code:        code,
				Name:        name,
				Description: description,
			})
			code, name, description = "", "", ""
		}
	}

	slices.SortFunc(codes, func(a, b HttpStatusCode) int {
		return strings.Compare(a.Code, b.Code)
	})
	return codes
}

func collectText(node *html.Node) string {
	var sb strings.Builder
	var text func(*html.Node)
	text = func(n *html.Node) {
		for cn := range n.ChildNodes() {
			if cn.Type == html.TextNode {
				if len(cn.Data) > 0 {
					sb.WriteString(cn.Data)
					continue
				}
			}

			if cn.Type == html.ElementNode {
				if cn.Data != "sup" {
					//skip citations
					text(cn)
				}
			}
		}
	}

	text(node)

	if sb.Len() == 0 {
		return strings.TrimSpace(node.Data)
	}
	return strings.TrimSpace(sb.String())
}
