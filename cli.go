package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/solerf/hst/local"
	"github.com/solerf/hst/pages"
	"github.com/solerf/hst/wikipedia"
)

const maxAgeLocal = 15 * 24 * time.Hour

type hst struct {
	StatusType string `optional:"" short:"t" long:"status-type" default:"" help:"HTTP status code types name to filter"`
	Code       string `optional:"" short:"c" long:"code" default:"" help:"HTTP status code to filter i.e:(401, 222, 1xx)"`
}

func (h *hst) Run() error {
	dir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get home dir: %w", err)
	}

	p, err := run(dir)
	if err != nil {
		return err
	}

	out, err := renderFiltered(p, h.StatusType, h.Code)
	if err != nil {
		return err
	}

	fmt.Println(out)
	return nil
}

func renderFiltered(p *pages.HttpStatusCodePage, statusType, code string) (string, error) {
	if statusType != "" {
		var ok bool
		p, ok = p.ByStatusType(statusType)
		if !ok {
			return "", fmt.Errorf("no status type matching %q", statusType)
		}
	}

	if code != "" {
		var ok bool
		p, ok = p.ByPrefixCode(code)

		if !ok {
			return "", fmt.Errorf("no status code matching %q", code)
		}
	}

	return p.ToJson()
}

func run(homeDir string) (*pages.HttpStatusCodePage, error) {
	localSrc, _ := local.SourceHTML(homeDir)

	if localSrc != nil && time.Since(localSrc.Date) <= maxAgeLocal {
		return localSrc, nil
	}

	remoteRev, err := reqWithTimeout(requestRevision)
	if err != nil {
		return nil, fmt.Errorf("impossible to load page revision: %w", err)
	}

	if localSrc != nil && localSrc.Revision >= remoteRev {
		localSrc.Date = time.Now()
		writeCache(homeDir, localSrc)
		return localSrc, nil
	}

	fresh, err := reqWithTimeout(requestHtml)
	if err != nil {
		return nil, fmt.Errorf("impossible to load page html: %w", err)
	}
	fresh.Revision = remoteRev
	writeCache(homeDir, fresh)

	return fresh, nil
}

func writeCache(homeDir string, p *pages.HttpStatusCodePage) {
	if err := local.Write(homeDir, p); err != nil {
		fmt.Fprintf(os.Stderr, "warning: cache write failed: %v\n", err)
	}
}

func reqWithTimeout[T any](f func(context.Context) (T, error)) (T, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return f(ctx)
}

func requestHtml(ctx context.Context) (*pages.HttpStatusCodePage, error) {
	h, err := wikipedia.SourceHTML(ctx, wikipedia.ListOfHttpStatusCodesUrl)
	if err != nil {
		return nil, err
	}
	return pages.ParseHttpStatusCodesPage(h, 0), nil
}

func requestRevision(ctx context.Context) (int, error) {
	return wikipedia.SourceRevision(ctx, wikipedia.RevisionOfHttpStatusCodesUrl)
}
