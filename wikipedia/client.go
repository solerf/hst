package wikipedia

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/net/html"
)

const ListOfHttpStatusCodesUrl = "https://en.wikipedia.org/w/rest.php/v1/page/List_of_HTTP_status_codes/html"

func SourceHTML(ctx context.Context, url string) (*html.Node, error) {
	request, err := http.NewRequestWithContext(ctx, "GET", url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("create request to %v: %w", url, err)
	}

	request.Header.Add("User-Agent", "any-project")

	deadline, ok := ctx.Deadline()
	timeout := time.Second * 5
	if ok {
		timeout = time.Until(deadline)
	}

	cli := http.Client{
		Timeout: timeout,
	}

	response, err := cli.Do(request)
	if err != nil {
		return nil, fmt.Errorf("requesting to %v: %w", url, err)
	}

	defer response.Body.Close()
	if response.StatusCode > 399 {
		return nil, fmt.Errorf("bad status %v: %v", response.Status, url)
	}

	node, err := html.Parse(response.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response %v: %w", url, err)
	}
	return node, nil
}
