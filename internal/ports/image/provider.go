package image

import (
	"context"

	domainimage "github.com/popododo0720/anarchy/internal/domain/image"
)

type Provider interface {
	ListImages(ctx context.Context) ([]domainimage.ImageSummary, error)
	GetImage(ctx context.Context, name string) (domainimage.ImageDetail, error)
}
