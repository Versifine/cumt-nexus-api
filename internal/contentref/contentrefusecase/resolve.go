package contentrefusecase

import (
	"context"
	"fmt"
	stdhtml "html"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/google/uuid"
	htmlparser "golang.org/x/net/html"
)

const (
	ProviderGenericLink  = "generic_link"
	ProviderBilibili     = "bilibili_video"
	ProviderDouyin       = "douyin_video"
	ProviderNeteaseMusic = "netease_music"
	ProviderQQMusic      = "qq_music"

	EmbedStatusReady       = "ready"
	EmbedStatusUnavailable = "unavailable"

	maxResolveInputLength = 4096
	maxResolveURLLength   = 2048
	maxRedirects          = 5
	maxMetadataBytes      = 512 * 1024
	defaultHTTPTimeout    = 4 * time.Second
)

var (
	httpURLPattern          = regexp.MustCompile(`(?i)https?://[^\s<>"'\x60]+`)
	schemelessURLPattern    = regexp.MustCompile("(?i)(v\\.douyin\\.com|open\\.douyin\\.com|www\\.douyin\\.com|douyin\\.com|iesdouyin\\.com|www\\.bilibili\\.com|bilibili\\.com|b23\\.tv|music\\.163\\.com|y\\.qq\\.com|c\\.y\\.qq\\.com|i\\.y\\.qq\\.com)(/[^\\s<>\"']*)?")
	bilibiliBVIDPattern     = regexp.MustCompile(`(?i)BV[0-9A-Za-z]{8,}`)
	bilibiliAVPathPattern   = regexp.MustCompile(`(?i)(?:^|/)video/av([0-9]+)(?:/|$)`)
	douyinVideoPattern      = regexp.MustCompile(`(?:^|/)video/([0-9]{5,64})(?:/|$)`)
	digitsPattern           = regexp.MustCompile(`^[0-9]+$`)
	tokenPattern            = regexp.MustCompile(`^[0-9A-Za-z_-]+$`)
	neteasePathRoutePattern = regexp.MustCompile(`^/(song|playlist|album)(?:/([0-9]+))?/?$`)
)

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type EmbedRepository interface {
	UpsertEmbed(ctx context.Context, embed Embed, now time.Time) (Embed, error)
}

type UseCase struct {
	embeds     EmbedRepository
	httpClient HTTPClient
	now        func() time.Time
}

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
	ID            string
	Provider      string
	ProviderRef   string
	URL           string
	CanonicalURL  string
	EmbedURL      string
	IframeAllowed bool
	Title         string
	Description   string
	ImageURL      string
	AuthorName    string
	Status        string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type embedMetadata struct {
	Title       string
	Description string
	ImageURL    string
	AuthorName  string
	Status      string
}

func NewUseCase(embedRepositories ...EmbedRepository) *UseCase {
	var embeds EmbedRepository
	if len(embedRepositories) > 0 {
		embeds = embedRepositories[0]
	}
	return &UseCase{
		embeds: embeds,
		httpClient: &http.Client{
			Timeout: defaultHTTPTimeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		now: time.Now,
	}
}

func (uc *UseCase) SetHTTPClient(client HTTPClient) {
	if client != nil {
		uc.httpClient = client
	}
}

func (uc *UseCase) SetNow(now func() time.Time) {
	if now != nil {
		uc.now = now
	}
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
	normalized, original, err := normalizePublicURL(input.URL)
	if err != nil {
		return Embed{}, err
	}

	normalized, err = uc.expandShortEmbedURL(ctx, normalized)
	if err != nil {
		return Embed{}, err
	}

	provider, providerRef, canonicalURL, embedURL, iframeAllowed, err := resolveEmbedProvider(normalized)
	if err != nil {
		return Embed{}, err
	}

	now := uc.now().UTC()
	embed := Embed{
		ID:            uuid.NewString(),
		Provider:      provider,
		ProviderRef:   providerRef,
		URL:           original,
		CanonicalURL:  canonicalURL,
		EmbedURL:      embedURL,
		IframeAllowed: iframeAllowed,
		Status:        EmbedStatusReady,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if uc.embeds != nil {
		metadata := uc.fetchEmbedMetadata(ctx, provider, canonicalURL)
		embed.Title = metadata.Title
		embed.Description = metadata.Description
		embed.ImageURL = metadata.ImageURL
		embed.AuthorName = metadata.AuthorName
		if metadata.Status != "" {
			embed.Status = metadata.Status
		}

		stored, err := uc.embeds.UpsertEmbed(ctx, embed, now)
		if err != nil {
			return Embed{}, fmt.Errorf("upsert embed: %w", err)
		}
		embed = stored
	}

	return embed, nil
}

func normalizePublicURL(rawInput string) (*url.URL, string, error) {
	rawURL, err := extractFirstPublicURL(rawInput)
	if err != nil {
		return nil, "", err
	}
	if len(rawURL) > maxResolveURLLength {
		return nil, "", apperr.New(apperr.CodeInvalidArgument, "url is too long")
	}

	parsed, err := url.Parse(rawURL)
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
	return &normalized, rawURL, nil
}

func extractFirstPublicURL(rawInput string) (string, error) {
	trimmed := strings.TrimSpace(rawInput)
	if trimmed == "" {
		return "", apperr.New(apperr.CodeInvalidArgument, "url is required")
	}
	if len(trimmed) > maxResolveInputLength {
		return "", apperr.New(apperr.CodeInvalidArgument, "url is too long")
	}

	if match := httpURLPattern.FindString(trimmed); match != "" {
		return trimURLPunctuation(match), nil
	}

	matches := schemelessURLPattern.FindStringSubmatch(trimmed)
	if len(matches) > 0 {
		return "https://" + trimURLPunctuation(matches[0]), nil
	}

	return "", apperr.New(apperr.CodeInvalidArgument, "url is invalid")
}

func trimURLPunctuation(rawURL string) string {
	return strings.TrimRight(rawURL, " \t\r\n，。；;、,.!?)]}）】》\"'")
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
	case hostMatches(host, "y.qq.com") || hostMatches(host, "c.y.qq.com") || hostMatches(host, "i.y.qq.com"):
		return ProviderQQMusic
	default:
		return ""
	}
}

func (uc *UseCase) expandShortEmbedURL(ctx context.Context, normalized *url.URL) (*url.URL, error) {
	if !shouldExpandShortURL(normalized.Hostname()) {
		return normalized, nil
	}

	current := normalized
	sourceHost := normalized.Hostname()
	for range maxRedirects {
		next, redirected, err := uc.followOneRedirect(ctx, current)
		if err != nil {
			return nil, err
		}
		if !redirected {
			return nil, apperr.New(apperr.CodeInvalidArgument, "short link cannot be expanded")
		}
		if !isAllowedShortURLRedirect(sourceHost, next.Hostname()) {
			return nil, apperr.New(apperr.CodeInvalidArgument, "short link redirects to unsupported host")
		}
		if !shouldExpandShortURL(next.Hostname()) {
			return next, nil
		}
		current = next
	}

	return nil, apperr.New(apperr.CodeInvalidArgument, "short link redirect limit exceeded")
}

func (uc *UseCase) followOneRedirect(ctx context.Context, current *url.URL) (*url.URL, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, current.String(), nil)
	if err != nil {
		return nil, false, apperr.New(apperr.CodeInvalidArgument, "short link is invalid")
	}
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	req.Header.Set("User-Agent", "cumt-nexus-api/1.0")

	resp, err := uc.httpClient.Do(req)
	if err != nil {
		return nil, false, apperr.New(apperr.CodeInvalidArgument, "short link cannot be expanded")
	}
	defer resp.Body.Close()

	if resp.StatusCode < 300 || resp.StatusCode > 399 {
		return nil, false, nil
	}

	location := strings.TrimSpace(resp.Header.Get("Location"))
	if location == "" {
		return nil, false, apperr.New(apperr.CodeInvalidArgument, "short link redirect is invalid")
	}

	parsedLocation, err := url.Parse(location)
	if err != nil {
		return nil, false, apperr.New(apperr.CodeInvalidArgument, "short link redirect is invalid")
	}
	next := current.ResolveReference(parsedLocation)
	normalized, _, err := normalizePublicURL(next.String())
	if err != nil {
		return nil, false, err
	}
	return normalized, true, nil
}

func shouldExpandShortURL(host string) bool {
	host = strings.TrimSuffix(strings.ToLower(host), ".")
	return host == "v.douyin.com" || host == "b23.tv"
}

func isAllowedShortURLRedirect(sourceHost string, targetHost string) bool {
	sourceHost = strings.TrimSuffix(strings.ToLower(sourceHost), ".")
	targetHost = strings.TrimSuffix(strings.ToLower(targetHost), ".")
	switch sourceHost {
	case "v.douyin.com":
		return hostMatches(targetHost, "douyin.com") || hostMatches(targetHost, "iesdouyin.com")
	case "b23.tv":
		return hostMatches(targetHost, "bilibili.com") || targetHost == "b23.tv"
	default:
		return false
	}
}

func resolveEmbedProvider(normalized *url.URL) (provider string, providerRef string, canonicalURL string, embedURL string, iframeAllowed bool, err error) {
	host := normalized.Hostname()
	switch {
	case hostMatches(host, "bilibili.com"):
		return resolveBilibiliEmbed(normalized)
	case host == "b23.tv":
		return "", "", "", "", false, apperr.New(apperr.CodeInvalidArgument, "bilibili canonical video url is required")
	case hostMatches(host, "douyin.com") || hostMatches(host, "iesdouyin.com"):
		return resolveDouyinEmbed(normalized)
	case hostMatches(host, "music.163.com"):
		return resolveNeteaseMusicEmbed(normalized)
	case hostMatches(host, "y.qq.com") || hostMatches(host, "c.y.qq.com") || hostMatches(host, "i.y.qq.com"):
		return resolveQQMusicEmbed(normalized)
	default:
		return "", "", "", "", false, apperr.New(apperr.CodeInvalidArgument, "embed provider is unsupported")
	}
}

func resolveBilibiliEmbed(normalized *url.URL) (string, string, string, string, bool, error) {
	bvid := bilibiliBVIDPattern.FindString(normalized.Path + " " + normalized.RawQuery)
	if strings.HasPrefix(strings.ToLower(bvid), "bv") && len(bvid) >= 2 {
		bvid = "BV" + bvid[2:]
	}
	aid := ""
	if bvid == "" {
		if matches := bilibiliAVPathPattern.FindStringSubmatch(normalized.Path); len(matches) >= 2 {
			aid = matches[1]
		}
		if aid == "" {
			aid = normalized.Query().Get("aid")
		}
		if aid == "" || !digitsPattern.MatchString(aid) {
			return "", "", "", "", false, apperr.New(apperr.CodeInvalidArgument, "bilibili video id is required")
		}
	}

	values := url.Values{}
	providerRef := bvid
	canonicalPath := "/video/" + bvid
	if bvid != "" {
		values.Set("bvid", bvid)
	} else {
		values.Set("aid", aid)
		providerRef = "av" + aid
		canonicalPath = "/video/av" + aid
	}
	if page := positiveIntegerQueryValue(normalized, "p"); page != "" {
		values.Set("p", page)
	}
	if start := positiveIntegerQueryValue(normalized, "t"); start != "" {
		values.Set("t", start)
	}
	values.Set("autoplay", "0")
	values.Set("danmaku", "0")
	embed := url.URL{
		Scheme:   "https",
		Host:     "player.bilibili.com",
		Path:     "/player.html",
		RawQuery: values.Encode(),
	}
	canonical := url.URL{
		Scheme: "https",
		Host:   "www.bilibili.com",
		Path:   canonicalPath,
	}
	return ProviderBilibili, providerRef, canonical.String(), embed.String(), true, nil
}

func resolveDouyinEmbed(normalized *url.URL) (string, string, string, string, bool, error) {
	videoID := douyinVideoID(normalized)
	if videoID == "" {
		return "", "", "", "", false, apperr.New(apperr.CodeInvalidArgument, "douyin video id is required")
	}
	canonical := url.URL{
		Scheme: "https",
		Host:   "www.douyin.com",
		Path:   "/video/" + videoID,
	}
	values := url.Values{}
	values.Set("vid", videoID)
	values.Set("autoplay", "0")
	embed := url.URL{
		Scheme:   "https",
		Host:     "open.douyin.com",
		Path:     "/player/video",
		RawQuery: values.Encode(),
	}
	return ProviderDouyin, videoID, canonical.String(), embed.String(), true, nil
}

func douyinVideoID(normalized *url.URL) string {
	if matches := douyinVideoPattern.FindStringSubmatch(normalized.Path); len(matches) >= 2 {
		return matches[1]
	}

	for _, key := range []string{"vid", "modal_id", "aweme_id", "video_id", "item_id"} {
		value := normalized.Query().Get(key)
		if isValidDouyinVideoID(value) {
			return value
		}
	}

	for _, key := range []string{"vid", "modal_id", "aweme_id", "video_id", "item_id"} {
		value := queryValueFromFragment(normalized.Fragment, key)
		if isValidDouyinVideoID(value) {
			return value
		}
	}

	return ""
}

func isValidDouyinVideoID(raw string) bool {
	raw = strings.TrimSpace(raw)
	return len(raw) >= 5 && len(raw) <= 64 && digitsPattern.MatchString(raw)
}

func resolveNeteaseMusicEmbed(normalized *url.URL) (string, string, string, string, bool, error) {
	resourceType, resourceID := neteaseMusicResource(normalized)
	if resourceID == "" || !digitsPattern.MatchString(resourceID) {
		return "", "", "", "", false, apperr.New(apperr.CodeInvalidArgument, "netease music id is required")
	}

	playerType, playerHeight := neteaseMusicPlayerConfig(resourceType)
	values := url.Values{}
	values.Set("type", playerType)
	values.Set("id", resourceID)
	values.Set("auto", "0")
	values.Set("height", playerHeight)
	embed := url.URL{
		Scheme:   "https",
		Host:     "music.163.com",
		Path:     "/outchain/player",
		RawQuery: values.Encode(),
	}
	canonical := url.URL{
		Scheme:   "https",
		Host:     "music.163.com",
		Path:     "/" + resourceType,
		RawQuery: url.Values{"id": []string{resourceID}}.Encode(),
	}
	return ProviderNeteaseMusic, resourceType + ":" + resourceID, canonical.String(), embed.String(), true, nil
}

func resolveQQMusicEmbed(normalized *url.URL) (string, string, string, string, bool, error) {
	songID := normalized.Query().Get("songid")
	songMid := normalized.Query().Get("songmid")
	if songMid == "" {
		songMid = normalized.Query().Get("mid")
	}
	if songMid == "" {
		songMid = songMidFromPath(normalized.Path)
	}

	values := url.Values{}
	providerRef := ""
	canonical := url.URL{}
	switch {
	case songID != "" && digitsPattern.MatchString(songID):
		values.Set("songid", songID)
		providerRef = "songid:" + songID
		canonical = url.URL{
			Scheme: "https",
			Host:   "i.y.qq.com",
			Path:   "/v8/playsong.html",
			RawQuery: url.Values{
				"songid": []string{songID},
			}.Encode(),
		}
	case songMid != "" && tokenPattern.MatchString(songMid):
		values.Set("songmid", songMid)
		providerRef = "songmid:" + songMid
		canonical = url.URL{
			Scheme: "https",
			Host:   "y.qq.com",
			Path:   "/n/ryqq/songDetail/" + songMid,
		}
	default:
		return "", "", "", "", false, apperr.New(apperr.CodeInvalidArgument, "qq music song id is required")
	}
	values.Set("songtype", firstNonEmpty(normalized.Query().Get("songtype"), "0"))

	embed := url.URL{
		Scheme:   "https",
		Host:     "i.y.qq.com",
		Path:     "/n2/m/outchain/player/index.html",
		RawQuery: values.Encode(),
	}
	return ProviderQQMusic, providerRef, canonical.String(), embed.String(), true, nil
}

func (uc *UseCase) fetchEmbedMetadata(ctx context.Context, provider string, canonicalURL string) embedMetadata {
	normalized, _, err := normalizePublicURL(canonicalURL)
	if err != nil {
		return embedMetadata{}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, normalized.String(), nil)
	if err != nil {
		return embedMetadata{}
	}
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	req.Header.Set("User-Agent", "cumt-nexus-api/1.0")

	resp, err := uc.httpClient.Do(req)
	if err != nil {
		return embedMetadata{}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone {
		return embedMetadata{Status: EmbedStatusUnavailable}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return embedMetadata{}
	}

	metadata := parseHTMLMetadata(io.LimitReader(resp.Body, maxMetadataBytes), normalized)
	if metadata.AuthorName == "" && provider == ProviderDouyin {
		metadata.AuthorName = deriveDouyinAuthorName(metadata.Title)
	}
	return metadata
}

func parseHTMLMetadata(reader io.Reader, baseURL *url.URL) embedMetadata {
	tokenizer := htmlparser.NewTokenizer(reader)
	metadata := embedMetadata{}

	for {
		tokenType := tokenizer.Next()
		switch tokenType {
		case htmlparser.ErrorToken:
			return metadata
		case htmlparser.StartTagToken, htmlparser.SelfClosingTagToken:
			token := tokenizer.Token()
			switch strings.ToLower(token.Data) {
			case "title":
				if metadata.Title == "" && tokenizer.Next() == htmlparser.TextToken {
					metadata.Title = cleanMetadataText(tokenizer.Token().Data, 200)
				}
			case "meta":
				applyMetaTag(&metadata, token, baseURL)
			}
		}
	}
}

func applyMetaTag(metadata *embedMetadata, token htmlparser.Token, baseURL *url.URL) {
	attrs := make(map[string]string, len(token.Attr))
	for _, attr := range token.Attr {
		attrs[strings.ToLower(strings.TrimSpace(attr.Key))] = strings.TrimSpace(attr.Val)
	}

	key := strings.ToLower(firstNonEmpty(attrs["property"], attrs["name"], attrs["itemprop"]))
	content := attrs["content"]
	if key == "" || content == "" {
		return
	}

	switch key {
	case "og:title", "twitter:title", "title":
		if metadata.Title == "" {
			metadata.Title = cleanMetadataText(content, 200)
		}
	case "description", "og:description", "twitter:description":
		if metadata.Description == "" {
			metadata.Description = cleanMetadataText(content, 500)
		}
	case "og:image", "twitter:image", "image":
		if metadata.ImageURL == "" {
			metadata.ImageURL = cleanMetadataURL(baseURL, content)
		}
	case "author", "article:author", "og:video:actor":
		if metadata.AuthorName == "" {
			metadata.AuthorName = cleanMetadataText(content, 80)
		}
	}
}

func cleanMetadataText(raw string, maxLength int) string {
	value := strings.Join(strings.Fields(stdhtml.UnescapeString(strings.TrimSpace(raw))), " ")
	runes := []rune(value)
	if len(runes) <= maxLength {
		return value
	}
	return string(runes[:maxLength])
}

func cleanMetadataURL(baseURL *url.URL, raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(stdhtml.UnescapeString(raw)))
	if err != nil {
		return ""
	}
	resolved := baseURL.ResolveReference(parsed)
	normalized, _, err := normalizePublicURL(resolved.String())
	if err != nil {
		return ""
	}
	return normalized.String()
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func deriveDouyinAuthorName(title string) string {
	title = strings.TrimSpace(title)
	for _, marker := range []string{"的抖音视频", "的作品"} {
		if index := strings.Index(title, marker); index > 0 && index <= 80 {
			return strings.TrimSpace(title[:index])
		}
	}
	return ""
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

func positiveIntegerQueryValue(normalized *url.URL, key string) string {
	value := normalized.Query().Get(key)
	if value == "" || !digitsPattern.MatchString(value) || strings.TrimLeft(value, "0") == "" {
		return ""
	}
	return value
}

func neteaseMusicResource(normalized *url.URL) (string, string) {
	if normalized.Path == "/outchain/player" {
		resourceID := normalized.Query().Get("id")
		switch normalized.Query().Get("type") {
		case "0":
			return "playlist", resourceID
		case "1":
			return "album", resourceID
		case "2", "":
			return "song", resourceID
		default:
			return "", ""
		}
	}

	if resourceType, resourceID := neteaseMusicResourceFromPath(normalized.Path, normalized.Query()); resourceType != "" {
		return resourceType, resourceID
	}

	if !strings.HasPrefix(normalized.Fragment, "/") {
		return "", ""
	}
	fragmentURL, err := url.Parse("https://music.163.com" + normalized.Fragment)
	if err != nil {
		return "", ""
	}
	return neteaseMusicResourceFromPath(fragmentURL.Path, fragmentURL.Query())
}

func neteaseMusicResourceFromPath(path string, values url.Values) (string, string) {
	matches := neteasePathRoutePattern.FindStringSubmatch(path)
	if len(matches) < 3 {
		return "", ""
	}
	resourceID := matches[2]
	if resourceID == "" {
		resourceID = values.Get("id")
	}
	return matches[1], resourceID
}

func neteaseMusicPlayerConfig(resourceType string) (string, string) {
	switch resourceType {
	case "playlist":
		return "0", "430"
	case "album":
		return "1", "430"
	default:
		return "2", "66"
	}
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
