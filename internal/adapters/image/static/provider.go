package static

import (
	"context"
	"fmt"

	domainimage "github.com/popododo0720/anarchy/internal/domain/image"
)

type Provider struct{}

func NewProvider() Provider {
	return Provider{}
}

func (Provider) ListImages(context.Context) ([]domainimage.ImageSummary, error) {
	return []domainimage.ImageSummary{{
		Name:       "ubuntu-24.04",
		SourceType: "local",
		Ready:      true,
		Size:       "2Gi",
	}}, nil
}

func (Provider) GetImage(ctx context.Context, name string) (domainimage.ImageDetail, error) {
	images, err := Provider{}.ListImages(ctx)
	if err != nil {
		return domainimage.ImageDetail{}, err
	}
	for _, image := range images {
		if image.Name == name {
			return domainimage.ImageDetail{
				Name:        image.Name,
				SourceType:  image.SourceType,
				Ready:       image.Ready,
				Size:        image.Size,
				Description: "Ubuntu image",
				Tags:        []string{"ubuntu", "24.04"},
			}, nil
		}
	}
	return domainimage.ImageDetail{}, fmt.Errorf("image not found: %s", name)
}
