package local

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/solerf/hst/pages"
)

func Write(homeDir string, p *pages.HttpStatusCodePage) (err error) {
	serialize, err := p.Serialize()
	if err != nil {
		return fmt.Errorf("serialize hst page: %w", err)
	}

	tmp, err := os.CreateTemp(homeDir, ".hst.tmp")
	if err != nil {
		return fmt.Errorf("create temp hst: %w", err)
	}
	tmpName := tmp.Name()

	defer func() {
		if err != nil {
			_ = os.Remove(tmpName)
		}
	}()

	if _, err = tmp.Write([]byte(serialize)); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temp hst: %w", err)
	}

	if err = tmp.Chmod(0600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("chmod temp hst: %w", err)
	}

	if err = tmp.Close(); err != nil {
		return fmt.Errorf("close temp hst: %w", err)
	}

	fName := filepath.Join(homeDir, ".hst")
	if err = os.Rename(tmpName, fName); err != nil {
		return fmt.Errorf("rename to %v: %w", fName, err)
	}

	return nil
}

func SourceHTML(homeDir string) (*pages.HttpStatusCodePage, error) {
	fName := filepath.Join(homeDir, ".hst")

	_, err := os.Stat(fName)
	if err != nil {
		return nil, fmt.Errorf("stat local hst: %w", err)
	}

	f, err := os.ReadFile(fName)
	if err != nil {
		return nil, fmt.Errorf("read local hst: %w", err)
	}

	return pages.Deserialize(f)
}
