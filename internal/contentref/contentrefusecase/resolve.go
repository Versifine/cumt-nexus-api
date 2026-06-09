package contentrefusecase

import (
	"context"
	"net"
	"net/netip"
	"net/url"
	"regexp"
	"strings"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
)

const (
	ProviderGenericLink  = "generic_link"
	ProviderBilibili     = "bilibili_video"
	ProviderDouyin       = "douyin_video"
	ProviderNeteaseMusic = "netease_music"
	ProviderQQMusic      = "qq_music"

	maxResolveURLLength = 2048
)

var (
	bilibiliBVIDPattern = regexp.MustCompile(`(?i)BV[0-9A-Za-z]+`)
	douyinVideoPattern  = regexp.MustCompile(`/video/([0-9]+)`)
	digitsPattern       = regexp.MustCompile(`^[0-9]+$`)
	tokenPattern        = regexp.MustCompile(`^[0-9A-Za-z_-]+$`)
)

type UseCase struct{}

type ResolveLinkPreviewInput struct {
	URL string
}

type ResolveEmbedInput struct {
	URL string
}

type LinkPreview struct {
	Provider     string
	URL          string
	CanonicalURL string
	Host         string
	Title        string
	Description  string
	ImageURL     string
}

type Embed struct {
	Provider      string
	URL           string
	CanonicalURL  string
	EmbedURL      string
	IframeAllowed bool
}

func NewUseCase() *UseCase {
	return &UseCase{}
}

func (uc *UseCase) ResolveLinkPreview(ctx context.Context, input ResolveLinkPreviewInput) (LinkPreview, error) {
	_ = ctx

	normalized, original, err := normalizePublicURL(input.URL)
	if err != nil {
		return LinkPreview{}, err
	}
	provider := detectProvider(normalized)
	if provider == "" {
		provider = ProviderGenericLink
	}

	return LinkPreview{
		Provider:     provider,
		URL:          original,
		CanonicalURL: normalized.String(),
		Host:         normalized.Hostname(),
		Title:        "",
		Description:  "",
		ImageURL:     "",
	}, nil
}

func (uc *UseCase) ResolveEmbed(ctx context.Context, input ResolveEmbedInput) (Embed, error) {
	_ = ctx

	normalized, original, err := normalizePublicURL(input.URL)
	if err != nil {
		return Embed{}, err
	}

	provider, embedURL, iframeAllowed, err := resolveEmbedProvider(normalized)
	if err != nil {
		return Embed{}, err
	}

	return Embed{
		Provider:      provider,
		URL:           original,
		CanonicalURL:  normalized.String(),
		EmbedURL:      embedURL,
		IframeAllowed: iframeAllowed,
	}, nil
}

func normalizePublicURL(rawURL string) (*url.URL, string, error) {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return nil, "", apperr.New(apperr.CodeInvalidArgument, "url is required")
	}
	if len(trimmed) > maxResolveURLLength {
		return nil, "", apperr.New(apperr.CodeInvalidArgument, "url is too long")
	}

	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, "", apperr.New(apperr.CodeInvalidArgument, "url is invalid")
	}
	if parsed.User != nil {
		return nil, "", apperr.New(apperr.CodeInvalidArgument, "url userinfo is not allowed")
	}

	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return nil, "", apperr.New(apperr.CodeInvalidArgument, "url scheme must be http or https")
	}

	host := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(parsed.Hostname())), ".")
	if host == "" || isBlockedHostname(host) {
		return nil, "", apperr.New(apperr.CodeInvalidArgument, "url host is not allowed")
	}
	port, err := parseURLPort(parsed)
	if err != nil {
		return nil, "", err
	}

	normalized := *parsed
	normalized.Scheme = scheme
	normalized.Host = buildURLHost(host, port)
	return &normalized, trimmed, nil
}

func parseURLPort(parsed *url.URL) (string, error) {
	port := parsed.Port()
	if port != "" {
		return port, nil
	}
	rawHost := parsed.Host
	if strings.Contains(rawHost, ":") {
		if strings.HasPrefix(rawHost, "[") && strings.HasSuffix(rawHost, "]") {
			return "", nil
		}
		return "", apperr.New(apperr.CodeInvalidArgument, "url port is invalid")
	}
	return "", nil
}

func isBlockedHostname(host string) bool {
	if host == "localhost" || strings.HasSuffix(host, ".localhost") || strings.HasSuffix(host, ".local") {
		return true
	}
	if strings.Contains(host, "%") {
		return true
	}

	if addr, err := netip.ParseAddr(host); err == nil {
		return addr.IsLoopback() ||
			addr.IsPrivate() ||
			addr.IsLinkLocalUnicast() ||
			addr.IsLinkLocalMulticast() ||
			addr.IsUnspecified() ||
			addr.IsMulticast()
	}

	return !strings.Contains(host, ".")
}

func buildURLHost(host string, port string) string {
	if port != "" {
		return net.JoinHostPort(host, port)
	}
	if strings.Contains(host, ":") {
		return "[" + host + "]"
	}
	return host
}

func detectProvider(normalized *url.URL) string {
	host := normalized.Hostname()
	switch {
	case hostMatches(host, "bilibili.com") || host == "b23.tv":
		return ProviderBilibili
	case hostMatches(host, "douyin.com") || hostMatches(host, "iesdouyin.com"):
		return ProviderDouyin
	case hostMatches(host, "music.163.com"):
		return ProviderNeteaseMusic
	case hostMatches(host, "y.qq.com") || hostMatches(host, "c.y.qq.com"):
		return ProviderQQMusic
	default:
		return ""
	}
}

func resolveEmbedProvider(normalized *url.URL) (provider string, embedURL string, iframeAllowed bool, err error) {
	host := normalized.Hostname()
	switch {
	case hostMatches(host, "bilibili.com"):
		return resolveBilibiliEmbed(normalized)
	case host == "b23.tv":
		return "", "", false, apperr.New(apperr.CodeInvalidArgument, "bilibili canonical video url is required")
	case hostMatches(host, "douyin.com") || hostMatches(host, "iesdouyin.com"):
		return resolveDouyinEmbed(normalized)
	case hostMatches(host, "music.163.com"):
		return resolveNeteaseMusicEmbed(normalized)
	case hostMatches(host, "y.qq.com") || hostMatches(host, "c.y.qq.com"):
		return resolveQQMusicEmbed(normalized)
	default:
		return "", "", false, apperr.New(apperr.CodeInvalidArgument, "embed provider is unsupported")
	}
}

func resolveBilibiliEmbed(normalized *url.URL) (string, string, bool, error) {
	bvid := bilibiliBVIDPattern.FindString(normalized.Path + " " + normalized.RawQuery)
	if bvid == "" {
		return "", "", false, apperr.New(apperr.CodeInvalidArgument, "bilibili video id is required")
	}
	if strings.HasPrefix(strings.ToLower(bvid), "bv") && len(bvid) >= 2 {
		bvid = "BV" + bvid[2:]
	}

	values := url.Values{}
	values.Set("bvid", bvid)
	embed := url.URL{
		Scheme:   "https",
		Host:     "player.bilibili.com",
		Path:     "/player.html",
		RawQuery: values.Encode(),
	}
	return ProviderBilibili, embed.String(), true, nil
}

func resolveDouyinEmbed(normalized *url.URL) (string, string, bool, error) {
	matches := douyinVideoPattern.FindStringSubmatch(normalized.Path)
	if len(matches) < 2 {
		return "", "", false, apperr.New(apperr.CodeInvalidArgument, "douyin video id is required")
	}
	videoID := matches[1]
	embed := url.URL{
		Scheme: "https",
		Host:   "www.douyin.com",
		Path:   "/video/" + videoID,
	}
	return ProviderDouyin, embed.String(), false, nil
}

func resolveNeteaseMusicEmbed(normalized *url.URL) (string, string, bool, error) {
	songID := normalized.Query().Get("id")
	if songID == "" {
		songID = queryValueFromFragment(normalized.Fragment, "id")
	}
	if songID == "" || !digitsPattern.MatchString(songID) {
		return "", "", false, apperr.New(apperr.CodeInvalidArgument, "netease music song id is required")
	}

	values := url.Values{}
	values.Set("type", "2")
	values.Set("id", songID)
	values.Set("auto", "0")
	values.Set("height", "66")
	embed := url.URL{
		Scheme:   "https",
		Host:     "music.163.com",
		Path:     "/outchain/player",
		RawQuery: values.Encode(),
	}
	return ProviderNeteaseMusic, embed.String(), true, nil
}

func resolveQQMusicEmbed(normalized *url.URL) (string, string, bool, error) {
	songMid := normalized.Query().Get("songmid")
	if songMid == "" {
		songMid = normalized.Query().Get("mid")
	}
	if songMid == "" {
		songMid = songMidFromPath(normalized.Path)
	}
	if songMid == "" || !tokenPattern.MatchString(songMid) {
		return "", "", false, apperr.New(apperr.CodeInvalidArgument, "qq music song id is required")
	}

	embed := url.URL{
		Scheme: "https",
		Host:   "y.qq.com",
		Path:   "/n/ryqq/songDetail/" + songMid,
	}
	return ProviderQQMusic, embed.String(), false, nil
}

func queryValueFromFragment(fragment string, key string) string {
	questionMark := strings.Index(fragment, "?")
	if questionMark < 0 || questionMark == len(fragment)-1 {
		return ""
	}
	values, err := url.ParseQuery(fragment[questionMark+1:])
	if err != nil {
		return ""
	}
	return values.Get(key)
}

func songMidFromPath(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	for index, part := range parts {
		if part == "songDetail" && index+1 < len(parts) {
			return parts[index+1]
		}
	}
	return ""
}

func hostMatches(host string, domain string) bool {
	host = strings.TrimSuffix(strings.ToLower(host), ".")
	domain = strings.ToLower(domain)
	return host == domain || strings.HasSuffix(host, "."+domain)
}
