package contentrefusecase

import (
	"context"
	"testing"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
)

func TestResolveLinkPreviewReturnsCanonicalPublicURL(t *testing.T) {
	uc := NewUseCase()

	result, err := uc.ResolveLinkPreview(context.Background(), ResolveLinkPreviewInput{
		URL: " HTTPS://Example.COM/path?q=1 ",
	})
	if err != nil {
		t.Fatalf("ResolveLinkPreview returned error: %v", err)
	}
	if result.Provider != ProviderGenericLink {
		t.Fatalf("expected provider %q, got %q", ProviderGenericLink, result.Provider)
	}
	if result.URL != "HTTPS://Example.COM/path?q=1" {
		t.Fatalf("unexpected original url: %q", result.URL)
	}
	if result.CanonicalURL != "https://example.com/path?q=1" || result.Host != "example.com" {
		t.Fatalf("unexpected canonical result: %#v", result)
	}
}

func TestResolveLinkPreviewRejectsBlockedHosts(t *testing.T) {
	uc := NewUseCase()

	blockedURLs := []string{
		"http://localhost:8080/a",
		"http://127.0.0.1/a",
		"http://10.0.0.1/a",
		"http://169.254.169.254/latest/meta-data",
		"http://intranet/a",
	}
	for _, rawURL := range blockedURLs {
		_, err := uc.ResolveLinkPreview(context.Background(), ResolveLinkPreviewInput{URL: rawURL})
		if !apperr.IsCode(err, apperr.CodeInvalidArgument) {
			t.Fatalf("expected invalid_argument for %q, got %v", rawURL, err)
		}
	}
}

func TestResolveLinkPreviewRejectsInvalidPort(t *testing.T) {
	uc := NewUseCase()

	_, err := uc.ResolveLinkPreview(context.Background(), ResolveLinkPreviewInput{
		URL: "https://example.com:bad/path",
	})
	if !apperr.IsCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument, got %v", err)
	}
}

func TestResolveEmbedReturnsBilibiliIframeURL(t *testing.T) {
	uc := NewUseCase()

	result, err := uc.ResolveEmbed(context.Background(), ResolveEmbedInput{
		URL: "https://www.bilibili.com/video/BV1xx411c7mD/?spm_id_from=333.1007",
	})
	if err != nil {
		t.Fatalf("ResolveEmbed returned error: %v", err)
	}
	if result.Provider != ProviderBilibili || !result.IframeAllowed {
		t.Fatalf("unexpected provider result: %#v", result)
	}
	if result.EmbedURL != "https://player.bilibili.com/player.html?bvid=BV1xx411c7mD" {
		t.Fatalf("unexpected embed url: %q", result.EmbedURL)
	}
}

func TestResolveEmbedReturnsNeteaseIframeURLFromFragment(t *testing.T) {
	uc := NewUseCase()

	result, err := uc.ResolveEmbed(context.Background(), ResolveEmbedInput{
		URL: "https://music.163.com/#/song?id=1901371647",
	})
	if err != nil {
		t.Fatalf("ResolveEmbed returned error: %v", err)
	}
	if result.Provider != ProviderNeteaseMusic || !result.IframeAllowed {
		t.Fatalf("unexpected provider result: %#v", result)
	}
	if result.EmbedURL != "https://music.163.com/outchain/player?auto=0&height=66&id=1901371647&type=2" {
		t.Fatalf("unexpected embed url: %q", result.EmbedURL)
	}
}

func TestResolveEmbedReturnsQQMusicCanonicalURL(t *testing.T) {
	uc := NewUseCase()

	result, err := uc.ResolveEmbed(context.Background(), ResolveEmbedInput{
		URL: "https://y.qq.com/n/ryqq/songDetail/0039MnYb0qxYhV",
	})
	if err != nil {
		t.Fatalf("ResolveEmbed returned error: %v", err)
	}
	if result.Provider != ProviderQQMusic || result.IframeAllowed {
		t.Fatalf("unexpected provider result: %#v", result)
	}
	if result.EmbedURL != "https://y.qq.com/n/ryqq/songDetail/0039MnYb0qxYhV" {
		t.Fatalf("unexpected embed url: %q", result.EmbedURL)
	}
}

func TestResolveEmbedRejectsUnsupportedProvider(t *testing.T) {
	uc := NewUseCase()

	_, err := uc.ResolveEmbed(context.Background(), ResolveEmbedInput{
		URL: "https://example.com/watch/1",
	})
	if !apperr.IsCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument, got %v", err)
	}
}
