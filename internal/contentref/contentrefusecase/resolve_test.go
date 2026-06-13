package contentrefusecase

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

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
		URL: "https://www.bilibili.com/video/BV1xx411c7mD/?spm_id_from=333.1007&p=2&t=30",
	})
	if err != nil {
		t.Fatalf("ResolveEmbed returned error: %v", err)
	}
	if result.Provider != ProviderBilibili || !result.IframeAllowed {
		t.Fatalf("unexpected provider result: %#v", result)
	}
	if result.ProviderRef != "BV1xx411c7mD" || result.CanonicalURL != "https://www.bilibili.com/video/BV1xx411c7mD" {
		t.Fatalf("unexpected canonical result: %#v", result)
	}
	if result.EmbedURL != "https://player.bilibili.com/player.html?autoplay=0&bvid=BV1xx411c7mD&danmaku=0&p=2&t=30" {
		t.Fatalf("unexpected embed url: %q", result.EmbedURL)
	}
}

func TestResolveEmbedReturnsBilibiliAIDIframeURL(t *testing.T) {
	uc := NewUseCase()

	result, err := uc.ResolveEmbed(context.Background(), ResolveEmbedInput{
		URL: "https://www.bilibili.com/video/av170001",
	})
	if err != nil {
		t.Fatalf("ResolveEmbed returned error: %v", err)
	}
	if result.ProviderRef != "av170001" || result.CanonicalURL != "https://www.bilibili.com/video/av170001" {
		t.Fatalf("unexpected canonical result: %#v", result)
	}
	if result.EmbedURL != "https://player.bilibili.com/player.html?aid=170001&autoplay=0&danmaku=0" {
		t.Fatalf("unexpected embed url: %q", result.EmbedURL)
	}
}

func TestResolveEmbedReturnsDouyinPlayerFromShareText(t *testing.T) {
	uc := NewUseCase()

	result, err := uc.ResolveEmbed(context.Background(), ResolveEmbedInput{
		URL: "9.99 复制打开抖音，看看这个视频 https://www.douyin.com/video/7123456789012345678?previous_page=app_code_link 更多内容",
	})
	if err != nil {
		t.Fatalf("ResolveEmbed returned error: %v", err)
	}
	if result.Provider != ProviderDouyin || !result.IframeAllowed {
		t.Fatalf("unexpected provider result: %#v", result)
	}
	if result.ProviderRef != "7123456789012345678" {
		t.Fatalf("unexpected provider ref: %q", result.ProviderRef)
	}
	if result.CanonicalURL != "https://www.douyin.com/video/7123456789012345678" {
		t.Fatalf("unexpected canonical url: %q", result.CanonicalURL)
	}
	if result.EmbedURL != "https://open.douyin.com/player/video?autoplay=0&vid=7123456789012345678" {
		t.Fatalf("unexpected embed url: %q", result.EmbedURL)
	}
}

func TestResolveEmbedReturnsDouyinPlayerFromModalID(t *testing.T) {
	uc := NewUseCase()

	result, err := uc.ResolveEmbed(context.Background(), ResolveEmbedInput{
		URL: "https://www.douyin.com/user/MS4wLjABAAAA?modal_id=7123456789012345678",
	})
	if err != nil {
		t.Fatalf("ResolveEmbed returned error: %v", err)
	}
	if result.ProviderRef != "7123456789012345678" {
		t.Fatalf("unexpected provider ref: %q", result.ProviderRef)
	}
	if result.EmbedURL != "https://open.douyin.com/player/video?autoplay=0&vid=7123456789012345678" {
		t.Fatalf("unexpected embed url: %q", result.EmbedURL)
	}
}

func TestResolveEmbedReturnsDouyinPlayerFromOpenPlayerURL(t *testing.T) {
	uc := NewUseCase()

	result, err := uc.ResolveEmbed(context.Background(), ResolveEmbedInput{
		URL: "open.douyin.com/player/video?vid=7123456789012345678&autoplay=0",
	})
	if err != nil {
		t.Fatalf("ResolveEmbed returned error: %v", err)
	}
	if result.ProviderRef != "7123456789012345678" {
		t.Fatalf("unexpected provider ref: %q", result.ProviderRef)
	}
}

func TestResolveEmbedExpandsDouyinShortLink(t *testing.T) {
	uc := NewUseCase()
	uc.SetHTTPClient(fakeHTTPClient{
		handler: func(req *http.Request) (*http.Response, error) {
			if req.URL.String() != "https://v.douyin.com/abc/" {
				t.Fatalf("unexpected request URL: %s", req.URL.String())
			}
			return textResponse(http.StatusFound, "", map[string]string{
				"Location": "https://www.douyin.com/video/7123456789012345678/",
			}), nil
		},
	})

	result, err := uc.ResolveEmbed(context.Background(), ResolveEmbedInput{
		URL: "https://v.douyin.com/abc/",
	})
	if err != nil {
		t.Fatalf("ResolveEmbed returned error: %v", err)
	}
	if result.ProviderRef != "7123456789012345678" || result.Provider != ProviderDouyin {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestResolveEmbedRejectsShortLinkToUnsupportedHost(t *testing.T) {
	uc := NewUseCase()
	uc.SetHTTPClient(fakeHTTPClient{
		handler: func(req *http.Request) (*http.Response, error) {
			return textResponse(http.StatusFound, "", map[string]string{
				"Location": "https://example.com/video/7123456789012345678",
			}), nil
		},
	})

	_, err := uc.ResolveEmbed(context.Background(), ResolveEmbedInput{
		URL: "https://v.douyin.com/abc/",
	})
	if !apperr.IsCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument, got %v", err)
	}
}

func TestResolveEmbedExpandsB23ShortLink(t *testing.T) {
	uc := NewUseCase()
	uc.SetHTTPClient(fakeHTTPClient{
		handler: func(req *http.Request) (*http.Response, error) {
			return textResponse(http.StatusFound, "", map[string]string{
				"Location": "https://www.bilibili.com/video/BV1xx411c7mD/",
			}), nil
		},
	})

	result, err := uc.ResolveEmbed(context.Background(), ResolveEmbedInput{
		URL: "b23.tv/abc",
	})
	if err != nil {
		t.Fatalf("ResolveEmbed returned error: %v", err)
	}
	if result.ProviderRef != "BV1xx411c7mD" || result.Provider != ProviderBilibili {
		t.Fatalf("unexpected result: %#v", result)
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
	if result.ProviderRef != "song:1901371647" {
		t.Fatalf("unexpected provider ref: %q", result.ProviderRef)
	}
	if result.EmbedURL != "https://music.163.com/outchain/player?auto=0&height=66&id=1901371647&type=2" {
		t.Fatalf("unexpected embed url: %q", result.EmbedURL)
	}
}

func TestResolveEmbedReturnsNeteasePlaylistIframeURL(t *testing.T) {
	uc := NewUseCase()

	result, err := uc.ResolveEmbed(context.Background(), ResolveEmbedInput{
		URL: "https://music.163.com/#/playlist?id=987654321",
	})
	if err != nil {
		t.Fatalf("ResolveEmbed returned error: %v", err)
	}
	if result.ProviderRef != "playlist:987654321" {
		t.Fatalf("unexpected provider ref: %q", result.ProviderRef)
	}
	if result.EmbedURL != "https://music.163.com/outchain/player?auto=0&height=430&id=987654321&type=0" {
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
	if result.Provider != ProviderQQMusic || !result.IframeAllowed {
		t.Fatalf("unexpected provider result: %#v", result)
	}
	if result.ProviderRef != "songmid:0039MnYb0qxYhV" {
		t.Fatalf("unexpected provider ref: %q", result.ProviderRef)
	}
	if result.EmbedURL != "https://i.y.qq.com/n2/m/outchain/player/index.html?songmid=0039MnYb0qxYhV&songtype=0" {
		t.Fatalf("unexpected embed url: %q", result.EmbedURL)
	}
}

func TestResolveEmbedReturnsQQMusicSongIDPlayerURL(t *testing.T) {
	uc := NewUseCase()

	result, err := uc.ResolveEmbed(context.Background(), ResolveEmbedInput{
		URL: "https://i.y.qq.com/v8/playsong.html?songid=102065756",
	})
	if err != nil {
		t.Fatalf("ResolveEmbed returned error: %v", err)
	}
	if result.ProviderRef != "songid:102065756" {
		t.Fatalf("unexpected provider ref: %q", result.ProviderRef)
	}
	if result.EmbedURL != "https://i.y.qq.com/n2/m/outchain/player/index.html?songid=102065756&songtype=0" {
		t.Fatalf("unexpected embed url: %q", result.EmbedURL)
	}
}

func TestResolveEmbedFetchesMetadataAndUpsertsEmbed(t *testing.T) {
	now := time.Date(2026, 6, 13, 10, 30, 0, 0, time.UTC)
	repo := &fakeEmbedRepository{}
	uc := NewUseCase(repo)
	uc.SetNow(func() time.Time { return now })
	uc.SetHTTPClient(fakeHTTPClient{
		handler: func(req *http.Request) (*http.Response, error) {
			if req.URL.String() != "https://www.douyin.com/video/7123456789012345678" {
				t.Fatalf("unexpected metadata URL: %s", req.URL.String())
			}
			return textResponse(http.StatusOK, `<!doctype html>
<html>
	<head>
		<meta property="og:title" content="张三的抖音视频">
		<meta name="description" content="校园生活记录">
		<meta property="og:image" content="/cover.jpg">
	</head>
</html>`, nil), nil
		},
	})

	result, err := uc.ResolveEmbed(context.Background(), ResolveEmbedInput{
		URL: "https://www.douyin.com/video/7123456789012345678",
	})
	if err != nil {
		t.Fatalf("ResolveEmbed returned error: %v", err)
	}
	if !repo.called {
		t.Fatal("expected embed repository to be called")
	}
	if repo.embed.Title != "张三的抖音视频" ||
		repo.embed.Description != "校园生活记录" ||
		repo.embed.ImageURL != "https://www.douyin.com/cover.jpg" ||
		repo.embed.AuthorName != "张三" ||
		repo.embed.Status != EmbedStatusReady {
		t.Fatalf("unexpected stored embed metadata: %#v", repo.embed)
	}
	if result.ID != repo.embed.ID || result.UpdatedAt != now {
		t.Fatalf("unexpected returned embed: %#v", result)
	}
}

func TestResolveEmbedMarksMetadata404Unavailable(t *testing.T) {
	repo := &fakeEmbedRepository{}
	uc := NewUseCase(repo)
	uc.SetHTTPClient(fakeHTTPClient{
		handler: func(req *http.Request) (*http.Response, error) {
			return textResponse(http.StatusNotFound, "", nil), nil
		},
	})

	result, err := uc.ResolveEmbed(context.Background(), ResolveEmbedInput{
		URL: "https://www.douyin.com/video/7123456789012345678",
	})
	if err != nil {
		t.Fatalf("ResolveEmbed returned error: %v", err)
	}
	if result.Status != EmbedStatusUnavailable || repo.embed.Status != EmbedStatusUnavailable {
		t.Fatalf("expected unavailable status, got result=%#v stored=%#v", result, repo.embed)
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

type fakeHTTPClient struct {
	handler func(req *http.Request) (*http.Response, error)
}

func (client fakeHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return client.handler(req)
}

type fakeEmbedRepository struct {
	called bool
	embed  Embed
}

func (repo *fakeEmbedRepository) UpsertEmbed(ctx context.Context, embed Embed, now time.Time) (Embed, error) {
	_ = ctx
	repo.called = true
	embed.CreatedAt = now
	embed.UpdatedAt = now
	repo.embed = embed
	return embed, nil
}

func textResponse(statusCode int, body string, headers map[string]string) *http.Response {
	response := &http.Response{
		StatusCode: statusCode,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
	for key, value := range headers {
		response.Header.Set(key, value)
	}
	return response
}
