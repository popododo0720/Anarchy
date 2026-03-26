package image

import (
	"context"

	domainimage "github.com/popododo0720/anarchy/internal/domain/image"
	portimage "github.com/popododo0720/anarchy/internal/ports/image"
)

type Service struct {
	provider portimage.Provider
}

func NewService(provider portimage.Provider) *Service {
	return &Service{provider: provider}
}

func (s *Service) ListImages(ctx context.Context) ([]domainimage.ImageSummary, error) {
	return s.provider.ListImages(ctx)
}

func (s *Service) GetImage(ctx context.Context, name string) (domainimage.ImageDetail, error) {
	return s.provider.GetImage(ctx, name)
}
