package main

import (
	"context"
	"fmt"
	"time"

	"github.com/solerf/hst/local"
	"github.com/solerf/hst/pages"
	"github.com/solerf/hst/wikipedia"
	"golang.org/x/net/html"
)

var cmd = hst{}

type hst struct {
	StatusType string `optional:"" short:"t" long:"statu-type" default:"" help:"HTTP status code types name to filter"`
	Code       string `optional:"" short:"c" long:"code" default:"" help:"HTTP status code to filter i.e:(401, 222, 1xx)"`
}

func (h *hst) Run() error {
	p, err := run()
	if err != nil {
		return err
	}

	if len(h.StatusType) != 0 {
		p = p.ByType(h.StatusType)
	}

	if len(h.Code) != 0 {
		p = p.ByCode(h.Code)
	}

	json, err := p.ToJson()
	if err != nil {
		return err
	}

	fmt.Println(json)
	return nil
}

func run() (*pages.HttpStatusCodePage, error) {
	localFile, _ := local.Source()
	h, rev, err := sourceHtmlWithRevision()

	if localFile == nil && err != nil {
		return nil, err
	}

	if localFile == nil || localFile.Revision < rev {
		localFile = pages.ParseHttpStatusCodesPage(h)
		if err = local.Write(localFile); err != nil {
			return nil, err
		}
	}

	return localFile, nil
}

func sourceHtmlWithRevision() (*html.Node, int, error) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelFunc()

	h, err := wikipedia.SourceHMTL(ctx, wikipedia.ListOfHttpStatusCodesUrl)
	if err != nil {
		return nil, 0, err
	}

	return h, pages.HttpStatusCodesPageRevision(h), nil
}
