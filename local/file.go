package local

import (
	"errors"
	"fmt"
	"os"

	"github.com/solerf/hst/pages"
)

func Write(p *pages.HttpStatusCodePage) error {
	homedir, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	serialize, err := p.Serialize()
	if err != nil {
		return fmt.Errorf("serialize hst %v: %v", p, err)
	}

	fName := fmt.Sprintf("%s/.hst", homedir)
	err = os.WriteFile(fName, []byte(serialize), 0666)
	if err != nil {
		return fmt.Errorf("write hst %v: %v", fName, err)
	}
	return nil
}

func SourceHTML() (*pages.HttpStatusCodePage, error) {
	homedir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("get home dir: %w", err)
	}

	fName := fmt.Sprintf("%s/.hst", homedir)
	if _, err = os.Stat(fName); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("stat local hst: %w", err)
	}

	f, err := os.ReadFile(fName)
	if err != nil {
		return nil, fmt.Errorf("read local hst: %w", err)
	}
	return pages.Deserialize(f)
}
