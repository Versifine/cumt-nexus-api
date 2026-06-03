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

function Add-Route {
    param(
        [System.Collections.Generic.Dictionary[string, object]]$Routes,
        [string]$Method,
        [string]$Path,
        [string]$Auth
    )

    $key = New-RouteKey -Method $Method -Path $Path
    if ($Routes.ContainsKey($key)) {
        throw "duplicate API route: $key"
    }
    $Routes[$key] = [pscustomobject]@{
        Method = $Method.ToUpperInvariant()
        Path = $Path
        Auth = $Auth
    }
}

function Add-RoutesFromFile {
    param(
        [System.Collections.Generic.Dictionary[string, object]]$Routes,
        [string]$Path,
        [string]$Prefix,
        [string]$Auth
    )

    $content = Get-Content -Raw -Encoding UTF8 -LiteralPath (Join-Path $repo $Path)
    $matches = [regex]::Matches($content, 'group\.(GET|POST|PUT|PATCH|DELETE)\("([^"]+)"')
    foreach ($match in $matches) {
        $method = $match.Groups[1].Value
        $routePath = $Prefix + $match.Groups[2].Value
        Add-Route -Routes $Routes -Method $method -Path $routePath -Auth $Auth
    }
}

$actualRoutes = [System.Collections.Generic.Dictionary[string, object]]::new([StringComparer]::Ordinal)

$healthContent = Get-Content -Raw -Encoding UTF8 -LiteralPath (Join-Path $repo 'internal/platform/httpserver/health.go')
if ([regex]::IsMatch($healthContent, 'router\.GET\("/healthz"')) {
    Add-Route -Routes $actualRoutes -Method 'GET' -Path '/healthz' -Auth 'public'
}

$mainContent = Get-Content -Raw -Encoding UTF8 -LiteralPath (Join-Path $repo 'cmd/api/main.go')
if ([regex]::IsMatch($mainContent, 'router\.Static\("/uploads"')) {
    Add-Route -Routes $actualRoutes -Method 'GET' -Path '/uploads/*filepath' -Auth 'public, local only'
}

Add-RoutesFromFile -Routes $actualRoutes -Path 'internal/auth/delivery/authhttp/register.go' -Prefix '/api/v1/auth' -Auth 'public'

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
    Add-RoutesFromFile -Routes $actualRoutes -Path $file -Prefix '/api/v1' -Auth 'Bearer'
}

if (-not (Test-Path -LiteralPath $docFullPath)) {
    throw "contract doc not found: $DocPath"
}

$documentedRoutes = [System.Collections.Generic.Dictionary[string, object]]::new([StringComparer]::Ordinal)
foreach ($line in Get-Content -Encoding UTF8 -LiteralPath $docFullPath) {
    $match = [regex]::Match($line, '^\|\s*(GET|POST|PUT|PATCH|DELETE)\s*\|\s*([^|]+?)\s*\|\s*([^|]+?)\s*\|')
    if ($match.Success) {
        $method = $match.Groups[1].Value
        $path = $match.Groups[2].Value.Trim()
        $auth = $match.Groups[3].Value.Trim()
        Add-Route -Routes $documentedRoutes -Method $method -Path $path -Auth $auth
    }
}

$missingInDoc = @($actualRoutes.Keys | Where-Object { -not $documentedRoutes.ContainsKey($_) } | Sort-Object)
$staleInDoc = @($documentedRoutes.Keys | Where-Object { -not $actualRoutes.ContainsKey($_) } | Sort-Object)
$authMismatches = @()

foreach ($routeKey in ($actualRoutes.Keys | Sort-Object)) {
    if (-not $documentedRoutes.ContainsKey($routeKey)) {
        continue
    }
    $actual = $actualRoutes[$routeKey]
    $documented = $documentedRoutes[$routeKey]
    if ($actual.Auth -ne $documented.Auth) {
        $authMismatches += [pscustomobject]@{
            route = $routeKey
            actual = $actual.Auth
            documented = $documented.Auth
        }
    }
}

if ($missingInDoc.Count -gt 0 -or $staleInDoc.Count -gt 0 -or $authMismatches.Count -gt 0) {
    [pscustomobject]@{
        status = 'failed'
        missing_in_doc = $missingInDoc
        stale_in_doc = $staleInDoc
        auth_mismatches = $authMismatches
    } | ConvertTo-Json -Depth 4
    throw 'API contract doc route/auth inventory is out of sync'
}

[pscustomobject]@{
    status = 'passed'
    route_count = $actualRoutes.Count
    doc = $DocPath
    routes = @($actualRoutes.Keys | Sort-Object)
    auth_boundaries = @($actualRoutes.Values | Sort-Object Method, Path)
} | ConvertTo-Json -Depth 4
