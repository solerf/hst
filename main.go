package main

import (
	"context"
	"fmt"
	"time"

	"github.com/solerf/hst/pages"
	"github.com/solerf/hst/wikipedia"
)

func main() {
	ctx, cancelFunc := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelFunc()

	html, err := wikipedia.SourceHMTL(ctx, wikipedia.ListOfHttpStatusCodesUrl)
	if err != nil {
		panic(err)
	}

	page := pages.ParseHttpStatusCodesPage(html)

	toJson, err := page.ToJson()
	if err != nil {
		panic(err)
	}
	fmt.Println(toJson)
}
