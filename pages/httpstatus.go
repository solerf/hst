package pages

import (
	"encoding/json"
	"iter"
	"path"
	"slices"
	"strconv"
	"strings"

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
	Revision  int
	CodeTypes []HttpStatusCodeType
}

func (h *HttpStatusCodePage) ToJson() (string, error) {
	j, err := json.MarshalIndent(h.CodeTypes, "  ", "    ")
	if err != nil {
		return "", err
	}
	return string(j), err
}

func ParseHttpStatusCodesPage(html *html.Node) *HttpStatusCodePage {
	return getHttpStatusList(html)
}

func getHttpStatusList(htmlNode *html.Node) *HttpStatusCodePage {
	var revision int
	codeTypes := make([]HttpStatusCodeType, 0, 11)

	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "html" {
			revision = collectRevision(node)
		}

		if node.Type == html.ElementNode && node.Data == "dl" {
			statusType := collectHttpStatusType(node)
			codes := collectHttpStatusCodes(node.ChildNodes())
			codeTypes = append(codeTypes, HttpStatusCodeType{
				Type:  statusType,
				Codes: codes,
			})
		}

		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}

	walk(htmlNode)

	slices.SortFunc(codeTypes, func(a, b HttpStatusCodeType) int {
		return strings.Compare(a.Type, b.Type)
	})

	return &HttpStatusCodePage{
		Revision:  revision,
		CodeTypes: codeTypes,
	}
}

func collectRevision(node *html.Node) int {
	for _, n := range node.Attr {
		if n.Key == "about" {
			revPath := path.Base(n.Val)
			// do nothing in case fails revision
			revision, _ := strconv.Atoi(revPath)
			return revision
		}
	}
	return 0
}

func collectHttpStatusType(node *html.Node) string {
	for n := range node.Parent.ChildNodes() {
		return collectText(n)
	}
	return ""
}

func collectHttpStatusCodes(nodes iter.Seq[*html.Node]) []HttpStatusCode {
	next, stop := iter.Pull(nodes)
	defer stop()

	codes := make([]HttpStatusCode, 0, 29)
	var code string
	var name string
	var description string

	// the nodes should be ordered at the html
	for {
		node, ok := next()
		if !ok {
			break
		}

		if node.Type != html.ElementNode {
			continue
		}

		switch node.Data {
		case "dt":
			text := strings.Split(collectText(node), " ")
			code, name = text[0], strings.Join(text[1:], "")
		case "dd":
			description = collectText(node)
		default:
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
	var text func(*html.Node, strings.Builder) string
	text = func(n *html.Node, b strings.Builder) string {
		for cn := range n.ChildNodes() {
			switch cn.Type {
			case html.TextNode:
				if len(cn.Data) > 0 {
					sb.WriteString(cn.Data)
					continue
				}
			case html.ElementNode:
				if cn.Data != "sup" {
					//skip citations
					text(cn, b)
				}
			default:
			}
		}

		if sb.Len() == 0 {
			return strings.TrimSpace(n.Data)
		}
		return strings.TrimSpace(sb.String())
	}
	return text(node, sb)
}
