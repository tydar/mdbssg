package host

import (
	"os"
	"path/filepath"
)

// Host describes a hosting solution for the static site
// to which generated static pages should be pushed
type Host interface {
	Save(text, slug, prefix string) error
}

// LocalHost provides an interface to saving files locally to the application for static site service
// path should point to the parent folder for all static files saved to this host
type LocalHost struct {
	path string
}

func NewLocalHost(path string) *LocalHost {
	return &LocalHost{
		path: path,
	}
}

// text: body of the HTML file
// slug: slug for the file
// prefix: any subdirectory structure under the lh.path parent folder
func (lh *LocalHost) Save(text, slug, prefix string) error {
	path := filepath.Join(lh.path, prefix)
	err := os.MkdirAll(path, os.ModePerm)
	if err != nil {
		return err
	}

	f, err := os.Create(filepath.Join(path, slug) + ".html")
	if err != nil {
		return err
	}

	f.WriteString(text)
	f.Close()
	return nil
}
