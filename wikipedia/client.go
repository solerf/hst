package wikipedia

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/net/html"
)

const ListOfHttpStatusCodesUrl = "https://en.wikipedia.org/w/rest.php/v1/page/List_of_HTTP_status_codes/html"
const RevisionOfHttpStatusCodesUrl = "https://en.wikipedia.org/w/rest.php/v1/page/List_of_HTTP_status_codes/bare"

func SourceHTML(ctx context.Context, url string) (*html.Node, error) {
	node, err := httpGet(ctx, url, html.Parse)
	if err != nil {
		return nil, fmt.Errorf("html source request: %w", err)
	}
	return node, nil
}

func SourceRevision(ctx context.Context, url string) (int, error) {
	decoder := func(r io.Reader) (int, error) {
		pb := &pageBare{}
		if err := json.NewDecoder(r).Decode(pb); err != nil {
			return 0, fmt.Errorf("decode revision: %w", err)
		}
		return pb.Latest.ID, nil
	}

	revision, err := httpGet(ctx, url, decoder)
	if err != nil {
		return 0, fmt.Errorf("revision request: %w", err)
	}
	return revision, nil
}

func httpGet[R any](ctx context.Context, url string, decoder func(io.Reader) (R, error)) (R, error) {
	var empty R

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return empty, fmt.Errorf("create request to %v: %w", url, err)
	}

	req.Header.Add("User-Agent", "any-project")
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return empty, fmt.Errorf("requesting to %v: %w", req.URL, err)
	}

	defer response.Body.Close()
	if response.StatusCode >= 400 {
		return empty, fmt.Errorf("bad status %v: %v", response.Status, req.URL)
	}

	result, err := decoder(response.Body)
	if err != nil {
		return empty, fmt.Errorf("reading response %v: %w", req.URL, err)
	}
	return result, nil
}

type pageBare struct {
	Latest struct {
		ID int `json:"id"`
	} `json:"latest"`
}
