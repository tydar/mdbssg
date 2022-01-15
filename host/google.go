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
	client *storage.Client
	bucket string
}

func NewGSHost(bucket string, client *storage.Client) *GSHost {
	return &GSHost{
		client: client,
		bucket: bucket,
	}
}

func (g *GSHost) Save(text, slug, prefix string) error {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 50*time.Second)
	defer cancel()

	wc := g.client.Bucket(g.bucket).Object(filepath.Join(prefix, slug) + ".html").NewWriter(ctx)
	if _, err := io.Copy(wc, strings.NewReader(text)); err != nil {
		return fmt.Errorf("io.Copy: %v", err)
	}

	if err := wc.Close(); err != nil {
		return fmt.Errorf("Writer.Close: %v", err)
	}
	log.Println("finished write")
	return nil
}
