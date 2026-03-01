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

const maxDaysLocal = 30

var cmd = hst{}

type hst struct {
	StatusType string `optional:"" short:"t" long:"status-type" default:"" help:"HTTP status code types name to filter"`
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
	localSrc, localRev, err := localHTMLWithRevision()

	isSourceRemote := err != nil || time.Now().Sub(localSrc.Date).Hours()/24 > maxDaysLocal

	if isSourceRemote {
		remoteSrc, remoteRev, remoteErr := requestHtmlWithRevision()
		if remoteErr != nil {
			return nil, fmt.Errorf("impossible to load status page: %w", remoteErr)
		}

		if localRev < remoteRev {
			localSrc = pages.ParseHttpStatusCodesPage(remoteSrc)
			// just ignore
			_ = local.Write(localSrc)
		}
	}

	return localSrc, nil
}

func localHTMLWithRevision() (*pages.HttpStatusCodePage, int, error) {
	var localSrc *pages.HttpStatusCodePage
	var err error

	localSrc, err = local.SourceHTML()
	if err != nil {
		return nil, 0, err
	}
	return localSrc, localSrc.Revision, nil
}

func requestHtmlWithRevision() (*html.Node, int, error) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelFunc()

	h, err := wikipedia.SourceHTML(ctx, wikipedia.ListOfHttpStatusCodesUrl)
	if err != nil {
		return nil, 0, err
	}

	return h, pages.HttpStatusCodesPageRevision(h), nil
}
