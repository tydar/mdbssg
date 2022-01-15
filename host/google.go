package host

import (
	"context"
	"fmt"
	"io"
	"log"
	"path/filepath"
	"strings"
	"time"

	"cloud.google.com/go/storage"
)

//GSHost represents a Google Cloud Storage hosting solution
type GSHost struct {
	bucket string
}

func NewGSHost(bucket string) *GSHost {
	return &GSHost{
		bucket: bucket,
	}
}

func (g *GSHost) Save(text, slug, prefix string) error {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("storage.NewClient: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(ctx, time.Second*50)
	defer cancel()

	wc := client.Bucket(g.bucket).Object(filepath.Join(prefix, slug) + ".html").NewWriter(ctx)
	if _, err = io.Copy(wc, strings.NewReader(text)); err != nil {
		return fmt.Errorf("io.Copy: %v", err)
	}

	if err := wc.Close(); err != nil {
		return fmt.Errorf("Writer.Close: %v", err)
	}
	log.Println("finished write")
	return nil
}
