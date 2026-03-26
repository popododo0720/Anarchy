package image_test

import (
	"context"
	"errors"
	"testing"

	appimage "github.com/popododo0720/anarchy/internal/application/image"
	domainimage "github.com/popododo0720/anarchy/internal/domain/image"
	portimage "github.com/popododo0720/anarchy/internal/ports/image"
)

type fakeProvider struct {
	images  []domainimage.ImageSummary
	image   domainimage.ImageDetail
	listErr error
	getErr  error
}

func (f fakeProvider) ListImages(context.Context) ([]domainimage.ImageSummary, error) {
	return f.images, f.listErr
}

func (f fakeProvider) GetImage(context.Context, string) (domainimage.ImageDetail, error) {
	return f.image, f.getErr
}

var _ portimage.Provider = fakeProvider{}

func TestServiceListImagesDelegatesToProvider(t *testing.T) {
	expected := []domainimage.ImageSummary{{Name: "ubuntu-24.04", Ready: true, SourceType: "local"}}
	service := appimage.NewService(fakeProvider{images: expected})

	got, err := service.ListImages(context.Background())
	if err != nil {
		t.Fatalf("ListImages() error = %v", err)
	}
	if len(got) != 1 || got[0].Name != "ubuntu-24.04" {
		t.Fatalf("ListImages() = %#v, want %#v", got, expected)
	}
}

func TestServiceGetImageDelegatesToProvider(t *testing.T) {
	expected := domainimage.ImageDetail{Name: "ubuntu-24.04", Ready: true, SourceType: "local"}
	service := appimage.NewService(fakeProvider{image: expected})

	got, err := service.GetImage(context.Background(), "ubuntu-24.04")
	if err != nil {
		t.Fatalf("GetImage() error = %v", err)
	}
	if got.Name != expected.Name || got.SourceType != expected.SourceType {
		t.Fatalf("GetImage() = %#v, want %#v", got, expected)
	}
}

func TestServiceListImagesReturnsProviderError(t *testing.T) {
	expectedErr := errors.New("boom")
	service := appimage.NewService(fakeProvider{listErr: expectedErr})

	_, err := service.ListImages(context.Background())
	if !errors.Is(err, expectedErr) {
		t.Fatalf("ListImages() error = %v, want %v", err, expectedErr)
	}
}
