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
		return err
	}

	fName := fmt.Sprintf("%s/.hst", homedir)
	return os.WriteFile(fName, []byte(serialize), 0666)
}

func Source() (*pages.HttpStatusCodePage, error) {
	homedir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	fName := fmt.Sprintf("%s/.hst", homedir)
	if _, err = os.Stat(fName); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	f, err := os.ReadFile(fName)
	if err != nil {
		return nil, err
	}
	return pages.Deserialize(f)
}
