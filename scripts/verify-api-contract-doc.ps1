param(
    [string]$DocPath = 'docs/contracts/http-api-contract.md'
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
        [string]$Auth,
        [string]$Handler = '',
        [string]$Source = ''
    )

    $key = New-RouteKey -Method $Method -Path $Path
    if ($Routes.ContainsKey($key)) {
        throw "duplicate API route: $key"
    }
    $Routes[$key] = [pscustomobject]@{
        Method = $Method.ToUpperInvariant()
        Path = $Path
        Auth = $Auth
        Handler = $Handler
        Source = $Source
    }
}

function Add-RoutesFromFile {
    param(
        [System.Collections.Generic.Dictionary[string, object]]$Routes,
        [string]$Path,
        [string]$Prefix,
        [string]$Auth,
        [string[]]$IncludeHandlers = @()
    )

    $content = Get-Content -Raw -Encoding UTF8 -LiteralPath (Join-Path $repo $Path)
    $matches = [regex]::Matches($content, 'group\.(GET|POST|PUT|PATCH|DELETE)\("([^"]+)",\s*handler\.(\w+)\)')
    foreach ($match in $matches) {
        $method = $match.Groups[1].Value
        $routePath = $Prefix + $match.Groups[2].Value
        $handler = $match.Groups[3].Value
        if ($IncludeHandlers.Count -gt 0 -and $IncludeHandlers -notcontains $handler) {
            continue
        }
        Add-Route -Routes $Routes -Method $method -Path $routePath -Auth $Auth -Handler $handler -Source $Path
    }
}

function Get-HandlerBody {
    param(
        [string]$Source,
        [string]$Handler
    )

    if ([string]::IsNullOrWhiteSpace($Source) -or [string]::IsNullOrWhiteSpace($Handler)) {
        return ''
    }

    $content = Get-Content -Raw -Encoding UTF8 -LiteralPath (Join-Path $repo $Source)
    $signature = [regex]::Match($content, "func\s+\(h \*Handler\)\s+$Handler\s*\(c \*gin\.Context\)\s*\{")
    if (-not $signature.Success) {
        return ''
    }

    $bodyStart = $signature.Index + $signature.Length
    $depth = 1
    for ($i = $bodyStart; $i -lt $content.Length; $i++) {
        $char = $content[$i]
        if ($char -eq '{') {
            $depth++
        } elseif ($char -eq '}') {
            $depth--
            if ($depth -eq 0) {
                return $content.Substring($bodyStart, $i - $bodyStart)
            }
        }
    }

    return ''
}

function Get-QueryParamsFromBody {
    param([string]$Body)

    $params = [System.Collections.Generic.HashSet[string]]::new([StringComparer]::Ordinal)
    foreach ($match in [regex]::Matches($Body, '\b(?:Query|DefaultQuery|GetQuery)\("([^"]+)"')) {
        [void]$params.Add($match.Groups[1].Value)
    }
    foreach ($match in [regex]::Matches($Body, '\b\w*Query\w*\(c,\s*"([^"]+)"')) {
        [void]$params.Add($match.Groups[1].Value)
    }
    return $params
}

function Read-QueryParamDocs {
    param([string]$Path)

    $docs = [System.Collections.Generic.Dictionary[string, object]]::new([StringComparer]::Ordinal)
    foreach ($line in Get-Content -Encoding UTF8 -LiteralPath $Path) {
        $match = [regex]::Match($line, '^\|\s*`([^`]+)`\s*\|\s*((?:`[^`]+`(?:,\s*)?)+)\s*\|')
        if (-not $match.Success) {
            continue
        }
        $routeKey = $match.Groups[1].Value.Trim()
        if ($docs.ContainsKey($routeKey)) {
            throw "duplicate API query parameter doc: $routeKey"
        }
        $params = @()
        foreach ($paramMatch in [regex]::Matches($match.Groups[2].Value, '`([^`]+)`')) {
            $params += $paramMatch.Groups[1].Value
        }
        $docs[$routeKey] = @($params | Sort-Object)
    }
    return $docs
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
    'internal/vote/delivery/votehttp/handler.go',
    'internal/moderation/delivery/moderationhttp/handler.go',
    'internal/notification/delivery/notificationhttp/handler.go',
    'internal/media/delivery/mediahttp/handler.go',
    'internal/contentref/delivery/contentrefhttp/handler.go'
)

foreach ($file in $protectedRouteFiles) {
    Add-RoutesFromFile -Routes $actualRoutes -Path $file -Prefix '/api/v1' -Auth 'Bearer'
}

Add-RoutesFromFile `
    -Routes $actualRoutes `
    -Path 'internal/user/delivery/userhttp/handler.go' `
    -Prefix '/api/v1' `
    -Auth 'optional Bearer' `
    -IncludeHandlers @('GetPublicUser')

Add-RoutesFromFile `
    -Routes $actualRoutes `
    -Path 'internal/user/delivery/userhttp/handler.go' `
    -Prefix '/api/v1' `
    -Auth 'Bearer' `
    -IncludeHandlers @('Me')

Add-RoutesFromFile `
    -Routes $actualRoutes `
    -Path 'internal/community/delivery/communityhttp/handler.go' `
    -Prefix '/api/v1' `
    -Auth 'optional Bearer' `
    -IncludeHandlers @('ListCommunities', 'GetCommunity')

Add-RoutesFromFile `
    -Routes $actualRoutes `
    -Path 'internal/community/delivery/communityhttp/handler.go' `
    -Prefix '/api/v1' `
    -Auth 'Bearer' `
    -IncludeHandlers @('SubmitCommunityApplication', 'ListCommunityApplications', 'GetCommunityApplication', 'ApproveCommunityApplication', 'RejectCommunityApplication', 'ListFollowedCommunities', 'FollowCommunity', 'DeleteCommunityFollow')

Add-RoutesFromFile `
    -Routes $actualRoutes `
    -Path 'internal/post/delivery/posthttp/handler.go' `
    -Prefix '/api/v1' `
    -Auth 'optional Bearer' `
    -IncludeHandlers @('ListCommunityPosts', 'ListLatestPosts', 'ListUserPosts', 'GetPost')

Add-RoutesFromFile `
    -Routes $actualRoutes `
    -Path 'internal/post/delivery/posthttp/handler.go' `
    -Prefix '/api/v1' `
    -Auth 'Bearer' `
    -IncludeHandlers @('PublishPost', 'ListSavedPosts', 'SavePost', 'DeletePostSave', 'UpdatePost', 'DeletePost')

Add-RoutesFromFile `
    -Routes $actualRoutes `
    -Path 'internal/comment/delivery/commenthttp/handler.go' `
    -Prefix '/api/v1' `
    -Auth 'optional Bearer' `
    -IncludeHandlers @('ListPostComments', 'ListUserComments')

Add-RoutesFromFile `
    -Routes $actualRoutes `
    -Path 'internal/search/delivery/searchhttp/handler.go' `
    -Prefix '/api/v1' `
    -Auth 'optional Bearer'

Add-RoutesFromFile `
    -Routes $actualRoutes `
    -Path 'internal/effect/delivery/effecthttp/handler.go' `
    -Prefix '/api/v1' `
    -Auth 'optional Bearer' `
    -IncludeHandlers @('ListEffectsCatalog')

Add-RoutesFromFile `
    -Routes $actualRoutes `
    -Path 'internal/comment/delivery/commenthttp/handler.go' `
    -Prefix '/api/v1' `
    -Auth 'Bearer' `
    -IncludeHandlers @('PublishComment', 'SetCommentVote', 'DeleteCommentVote', 'UpdateComment', 'DeleteComment')

Add-RoutesFromFile `
    -Routes $actualRoutes `
    -Path 'internal/effect/delivery/effecthttp/handler.go' `
    -Prefix '/api/v1' `
    -Auth 'Bearer' `
    -IncludeHandlers @('GetMyPoints', 'ApplyCommentEffect')

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
$actualQueryParamDocs = [System.Collections.Generic.Dictionary[string, object]]::new([StringComparer]::Ordinal)
$documentedQueryParamDocs = Read-QueryParamDocs -Path $docFullPath
$queryParamMismatches = @()

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

    $body = Get-HandlerBody -Source $actual.Source -Handler $actual.Handler
    $queryParams = Get-QueryParamsFromBody -Body $body
    if ($queryParams.Count -gt 0) {
        $actualQueryParamDocs[$routeKey] = @($queryParams | Sort-Object)
    }
}

$missingQueryParamDocs = @($actualQueryParamDocs.Keys | Where-Object { -not $documentedQueryParamDocs.ContainsKey($_) } | Sort-Object)
$staleQueryParamDocs = @($documentedQueryParamDocs.Keys | Where-Object { -not $actualQueryParamDocs.ContainsKey($_) } | Sort-Object)
foreach ($routeKey in ($actualQueryParamDocs.Keys | Sort-Object)) {
    if (-not $documentedQueryParamDocs.ContainsKey($routeKey)) {
        continue
    }
    $actualParams = @($actualQueryParamDocs[$routeKey] | Sort-Object)
    $documentedParams = @($documentedQueryParamDocs[$routeKey] | Sort-Object)
    if (($actualParams -join ',') -ne ($documentedParams -join ',')) {
        $queryParamMismatches += [pscustomobject]@{
            route = $routeKey
            actual = $actualParams
            documented = $documentedParams
        }
    }
}

if (
    $missingInDoc.Count -gt 0 -or
    $staleInDoc.Count -gt 0 -or
    $authMismatches.Count -gt 0 -or
    $missingQueryParamDocs.Count -gt 0 -or
    $staleQueryParamDocs.Count -gt 0 -or
    $queryParamMismatches.Count -gt 0
) {
    [pscustomobject]@{
        status = 'failed'
        missing_in_doc = $missingInDoc
        stale_in_doc = $staleInDoc
        auth_mismatches = $authMismatches
        missing_query_param_docs = $missingQueryParamDocs
        stale_query_param_docs = $staleQueryParamDocs
        query_param_mismatches = $queryParamMismatches
    } | ConvertTo-Json -Depth 4
    throw 'API contract doc route/auth/query inventory is out of sync'
}

[pscustomobject]@{
    status = 'passed'
    route_count = $actualRoutes.Count
    query_param_route_count = $actualQueryParamDocs.Count
    doc = $DocPath
    routes = @($actualRoutes.Keys | Sort-Object)
    auth_boundaries = @($actualRoutes.Values | Sort-Object Method, Path | ForEach-Object {
        [pscustomobject]@{
            Method = $_.Method
            Path = $_.Path
            Auth = $_.Auth
        }
    })
    query_params = @($actualQueryParamDocs.Keys | Sort-Object | ForEach-Object {
        [pscustomobject]@{
            route = $_
            params = @($actualQueryParamDocs[$_])
        }
    })
} | ConvertTo-Json -Depth 4
