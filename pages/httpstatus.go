package pages

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
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
	Revision  int                  `json:"revision"`
	CodeTypes []HttpStatusCodeType `json:"code_types"`
}

func (h *HttpStatusCodePage) ByType(t string) *HttpStatusCodePage {
	idx := slices.IndexFunc(h.CodeTypes, func(ct HttpStatusCodeType) bool {
		return strings.Contains(strings.ToLower(ct.Type), strings.ToLower(t))
	})

	if idx != -1 {
		h.CodeTypes = h.CodeTypes[idx : idx+1]
	}
	return h
}

func (h *HttpStatusCodePage) ByCode(c string) *HttpStatusCodePage {
	for codeTypeIdx, codeType := range h.CodeTypes {
		idx := slices.IndexFunc(codeType.Codes, func(s HttpStatusCode) bool {
			return strings.ToLower(s.Code) == strings.ToLower(c)
		})

		if idx != -1 {
			h.CodeTypes = h.CodeTypes[codeTypeIdx : codeTypeIdx+1]
			h.CodeTypes[0].Codes = h.CodeTypes[0].Codes[idx : idx+1]
			break
		}
	}
	return h
}

func (h *HttpStatusCodePage) ToJson() (string, error) {
	j, err := json.MarshalIndent(h.CodeTypes, " ", "  ")
	if err != nil {
		return "", err
	}
	return string(j), err
}

func (h *HttpStatusCodePage) Serialize() (string, error) {
	in, err := json.Marshal(h)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(in), nil
}

func Deserialize(in []byte) (*HttpStatusCodePage, error) {
	if len(in) != 0 {
		out, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(in)))
		if err != nil {
			return nil, err
		}

		var h *HttpStatusCodePage
		if err = json.Unmarshal(out, h); err != nil {
			return nil, err
		}
		return h, nil
	}
	return nil, errors.New("empty bytes to deserialize")
}

func ParseHttpStatusCodesPage(html *html.Node) *HttpStatusCodePage {
	return getHttpStatusList(html)
}

func HttpStatusCodesPageRevision(html *html.Node) int {
	return getRevisionOnly(html)
}

func getRevisionOnly(htmlNode *html.Node) int {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for node := range streamHtmlNodes(ctx, htmlNode) {
		if node.Type == html.ElementNode && node.Data == "html" {
			return collectRevision(node)
		}
	}
	return 0
}

func getHttpStatusList(htmlNode *html.Node) *HttpStatusCodePage {
	var revision int
	codeTypes := make([]HttpStatusCodeType, 0, 11)

	for node := range streamHtmlNodes(context.Background(), htmlNode) {
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
	}

	slices.SortFunc(codeTypes, func(a, b HttpStatusCodeType) int {
		return strings.Compare(a.Type, b.Type)
	})

	return &HttpStatusCodePage{
		Revision:  revision,
		CodeTypes: codeTypes,
	}
}

func streamHtmlNodes(ctx context.Context, htmlNode *html.Node) <-chan *html.Node {
	nodeChan := make(chan *html.Node, 1)

	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode {
			nodeChan <- node
		}

		var child *html.Node
		for child = node.FirstChild; child != nil; child = child.NextSibling {
			select {
			case <-ctx.Done():
				close(nodeChan)
				return
			default:
				walk(child)
			}
		}
	}

	go func() {
		walk(htmlNode)
		close(nodeChan)
	}()
	return nodeChan
}

func collectRevision(node *html.Node) int {
	for _, n := range node.Attr {
		if n.Key == "about" {
			revPath := path.Base(n.Val)
			revision, err := strconv.Atoi(revPath)
			if err != nil {
				// do nothing if fails, just skip it
				break
			}
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
