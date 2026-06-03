param(
    [string]$DocPath = 'docs/internal/architecture/http-api-contract.md'
)

$ErrorActionPreference = 'Stop'

$repo = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$docFullPath = Join-Path $repo $DocPath

function New-RouteKey {
    param(
        [string]$Method,
        [string]$Path
    )
    return "$($Method.ToUpperInvariant()) $Path"
}

function Add-RoutesFromFile {
    param(
        [System.Collections.Generic.HashSet[string]]$Routes,
        [string]$Path,
        [string]$Prefix
    )

    $content = Get-Content -Raw -Encoding UTF8 -LiteralPath (Join-Path $repo $Path)
    $matches = [regex]::Matches($content, 'group\.(GET|POST|PUT|PATCH|DELETE)\("([^"]+)"')
    foreach ($match in $matches) {
        $method = $match.Groups[1].Value
        $routePath = $Prefix + $match.Groups[2].Value
        [void]$Routes.Add((New-RouteKey -Method $method -Path $routePath))
    }
}

$actualRoutes = [System.Collections.Generic.HashSet[string]]::new([StringComparer]::Ordinal)

$healthContent = Get-Content -Raw -Encoding UTF8 -LiteralPath (Join-Path $repo 'internal/platform/httpserver/health.go')
if ([regex]::IsMatch($healthContent, 'router\.GET\("/healthz"')) {
    [void]$actualRoutes.Add((New-RouteKey -Method 'GET' -Path '/healthz'))
}

$mainContent = Get-Content -Raw -Encoding UTF8 -LiteralPath (Join-Path $repo 'cmd/api/main.go')
if ([regex]::IsMatch($mainContent, 'router\.Static\("/uploads"')) {
    [void]$actualRoutes.Add((New-RouteKey -Method 'GET' -Path '/uploads/*filepath'))
}

Add-RoutesFromFile -Routes $actualRoutes -Path 'internal/auth/delivery/authhttp/register.go' -Prefix '/api/v1/auth'

$protectedRouteFiles = @(
    'internal/user/delivery/userhttp/handler.go',
    'internal/community/delivery/communityhttp/handler.go',
    'internal/post/delivery/posthttp/handler.go',
    'internal/comment/delivery/commenthttp/handler.go',
    'internal/vote/delivery/votehttp/handler.go',
    'internal/moderation/delivery/moderationhttp/handler.go',
    'internal/search/delivery/searchhttp/handler.go',
    'internal/notification/delivery/notificationhttp/handler.go',
    'internal/media/delivery/mediahttp/handler.go'
)

foreach ($file in $protectedRouteFiles) {
    Add-RoutesFromFile -Routes $actualRoutes -Path $file -Prefix '/api/v1'
}

if (-not (Test-Path -LiteralPath $docFullPath)) {
    throw "contract doc not found: $DocPath"
}

$documentedRoutes = [System.Collections.Generic.HashSet[string]]::new([StringComparer]::Ordinal)
foreach ($line in Get-Content -Encoding UTF8 -LiteralPath $docFullPath) {
    $match = [regex]::Match($line, '^\|\s*(GET|POST|PUT|PATCH|DELETE)\s*\|\s*([^|]+?)\s*\|')
    if ($match.Success) {
        $method = $match.Groups[1].Value
        $path = $match.Groups[2].Value.Trim()
        [void]$documentedRoutes.Add((New-RouteKey -Method $method -Path $path))
    }
}

$missingInDoc = @($actualRoutes | Where-Object { -not $documentedRoutes.Contains($_) } | Sort-Object)
$staleInDoc = @($documentedRoutes | Where-Object { -not $actualRoutes.Contains($_) } | Sort-Object)

if ($missingInDoc.Count -gt 0 -or $staleInDoc.Count -gt 0) {
    [pscustomobject]@{
        status = 'failed'
        missing_in_doc = $missingInDoc
        stale_in_doc = $staleInDoc
    } | ConvertTo-Json -Depth 4
    throw 'API contract doc route inventory is out of sync'
}

[pscustomobject]@{
    status = 'passed'
    route_count = $actualRoutes.Count
    doc = $DocPath
    routes = @($actualRoutes | Sort-Object)
} | ConvertTo-Json -Depth 4
