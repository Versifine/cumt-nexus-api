param(
    [int]$Port = 18080,
    [switch]$SkipMigration
)

$ErrorActionPreference = 'Stop'

function Assert-True {
    param(
        [bool]$Condition,
        [string]$Message
    )
    if (-not $Condition) {
        throw $Message
    }
}

function Assert-HttpErrorCode {
    param(
        [scriptblock]$Request,
        [int]$StatusCode,
        [string]$ErrorCode
    )

    try {
        & $Request | Out-Null
        throw "expected HTTP $StatusCode with error code $ErrorCode, got success"
    } catch {
        $response = $_.Exception.Response
        if ($null -eq $response) {
            throw
        }
        if ([int]$response.StatusCode -ne $StatusCode) {
            throw "expected HTTP $StatusCode, got $([int]$response.StatusCode)"
        }

        $body = $null
        if ($_.ErrorDetails -and $_.ErrorDetails.Message) {
            $body = $_.ErrorDetails.Message | ConvertFrom-Json
        } else {
            $stream = $response.GetResponseStream()
            $reader = [System.IO.StreamReader]::new($stream)
            $body = ($reader.ReadToEnd()) | ConvertFrom-Json
        }
        if ($body.error.code -ne $ErrorCode) {
            throw "expected error code $ErrorCode, got $($body.error.code)"
        }
    }
}

$repo = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$smokeRoot = Join-Path $repo (Join-Path '.tmp' ('s13-smoke-' + [guid]::NewGuid().ToString('N')))
$apiExe = Join-Path $smokeRoot 'api-smoke.exe'
$uploads = Join-Path $smokeRoot 'uploads'
$job = $null

try {
    New-Item -ItemType Directory -Force -Path $uploads | Out-Null

    if (-not $SkipMigration) {
        Push-Location $repo
        try {
            go run ./cmd/migrate up
        } finally {
            Pop-Location
        }
    }

    Push-Location $repo
    try {
        go build -buildvcs=false -o $apiExe ./cmd/api
    } finally {
        Pop-Location
    }

    $baseURL = "http://127.0.0.1:$Port"
    $job = Start-Job -ScriptBlock {
        param($RepoPath, $ExePath, $UploadPath, $ListenPort)

        Set-Location $RepoPath
        $env:APP_NAME = 'cumt-nexus-api-smoke'
        $env:APP_ENV = 'local'
        $env:APP_STARTUP_TIMEOUT = '10s'
        $env:HTTP_ADDR = "127.0.0.1:$ListenPort"
        $env:AUTH_TOKEN_SECRET = 'SmokeAuthSecretForStage13'
        $env:OBJECT_STORAGE_PROVIDER = 'local'
        $env:OBJECT_STORAGE_PUBLIC_BASE_URL = "http://127.0.0.1:$ListenPort/uploads"
        $env:OBJECT_STORAGE_LOCAL_ROOT = $UploadPath
        $env:UPLOAD_IMAGE_MAX_BYTES = '5242880'
        $env:UPLOAD_IMAGE_MAX_COUNT_PER_POST = '9'
        $env:UPLOAD_IMAGE_MAX_COUNT_PER_COMMENT = '1'

        & $ExePath
    } -ArgumentList $repo, $apiExe, $uploads, $Port

    $ready = $false
    for ($i = 0; $i -lt 60; $i++) {
        if ($job.State -ne 'Running') {
            $jobOutput = Receive-Job $job -Keep | Out-String
            throw "API job stopped before readiness: state=$($job.State) output=$jobOutput"
        }

        try {
            $health = Invoke-RestMethod -Uri "$baseURL/healthz" -Method Get -TimeoutSec 2
            if ($health.status -eq 'ok') {
                $ready = $true
                break
            }
        } catch {
            Start-Sleep -Milliseconds 500
        }
    }
    Assert-True $ready "API did not become ready at $baseURL"

    $suffix = Get-Date -Format 'yyMMddHHmmssfff'
    $username = "s13_smoke_$suffix"
    $registerBody = @{ username = $username; password = 'password123' } | ConvertTo-Json -Compress
    $register = Invoke-RestMethod -Uri "$baseURL/api/v1/auth/register" -Method Post -ContentType 'application/json' -Body $registerBody
    $token = $register.access_token
    Assert-True (-not [string]::IsNullOrWhiteSpace($token)) 'register did not return access_token'
    $headers = @{ Authorization = "Bearer $token" }

    $pngPath = Join-Path $smokeRoot 'smoke.png'
    [IO.File]::WriteAllBytes($pngPath, [Convert]::FromBase64String('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+/p9sAAAAASUVORK5CYII='))

    $uploadPostRaw = & curl.exe -sS -X POST "$baseURL/api/v1/uploads/images" -H "Authorization: Bearer $token" -F "file=@$pngPath" -F "alt_text=post smoke"
    if ($LASTEXITCODE -ne 0) {
        throw "curl post upload failed: $uploadPostRaw"
    }
    $uploadPost = $uploadPostRaw | ConvertFrom-Json
    $postAttachmentID = $uploadPost.attachment.id
    Assert-True (-not [string]::IsNullOrWhiteSpace($postAttachmentID)) "post upload missing attachment id: $uploadPostRaw"

    $postBody = @{
        title = 'Stage 13 smoke post'
        body = 'Markdown-like post body'
        attachment_ids = @($postAttachmentID)
    } | ConvertTo-Json -Compress
    $post = Invoke-RestMethod -Uri "$baseURL/api/v1/communities/public/posts" -Method Post -Headers $headers -ContentType 'application/json' -Body $postBody
    $postID = $post.post.id
    Assert-True (-not [string]::IsNullOrWhiteSpace($postID)) 'post create did not return id'
    Assert-True ($post.post.body_format -eq 'markdown') "expected post body_format markdown, got $($post.post.body_format)"
    Assert-True ($post.post.attachments.Count -eq 1) "post create did not return one attachment"

    $uploadRootRaw = & curl.exe -sS -X POST "$baseURL/api/v1/uploads/images" -H "Authorization: Bearer $token" -F "file=@$pngPath" -F "alt_text=root comment smoke"
    if ($LASTEXITCODE -ne 0) {
        throw "curl root comment upload failed: $uploadRootRaw"
    }
    $uploadRoot = $uploadRootRaw | ConvertFrom-Json
    $rootAttachmentID = $uploadRoot.attachment.id
    Assert-True (-not [string]::IsNullOrWhiteSpace($rootAttachmentID)) "root comment upload missing attachment id: $uploadRootRaw"

    $rootBody = @{
        body = 'Markdown-like root comment body'
        attachment_ids = @($rootAttachmentID)
    } | ConvertTo-Json -Compress
    $root = Invoke-RestMethod -Uri "$baseURL/api/v1/posts/$postID/comments" -Method Post -Headers $headers -ContentType 'application/json' -Body $rootBody
    $rootCommentID = $root.comment.id
    Assert-True (-not [string]::IsNullOrWhiteSpace($rootCommentID)) 'root comment create did not return id'
    Assert-True ($root.comment.attachments.Count -eq 1) 'root comment create did not return one attachment'

    $uploadChildRaw = & curl.exe -sS -X POST "$baseURL/api/v1/uploads/images" -H "Authorization: Bearer $token" -F "file=@$pngPath" -F "alt_text=child comment smoke"
    if ($LASTEXITCODE -ne 0) {
        throw "curl child comment upload failed: $uploadChildRaw"
    }
    $uploadChild = $uploadChildRaw | ConvertFrom-Json
    $childAttachmentID = $uploadChild.attachment.id
    Assert-True (-not [string]::IsNullOrWhiteSpace($childAttachmentID)) "child comment upload missing attachment id: $uploadChildRaw"

    $childBody = @{
        body = 'Markdown-like child comment body'
        parent_id = $rootCommentID
        attachment_ids = @($childAttachmentID)
    } | ConvertTo-Json -Compress
    $child = Invoke-RestMethod -Uri "$baseURL/api/v1/posts/$postID/comments" -Method Post -Headers $headers -ContentType 'application/json' -Body $childBody
    $childCommentID = $child.comment.id
    Assert-True (-not [string]::IsNullOrWhiteSpace($childCommentID)) 'child comment create did not return id'
    Assert-True ($child.comment.attachments.Count -eq 1) 'child comment create did not return one attachment'

    $tree = Invoke-RestMethod -Uri "$baseURL/api/v1/posts/$postID/comments?view=tree&sort=new&limit=20&offset=0&max_depth=6" -Method Get -Headers $headers
    Assert-True ($tree.view -eq 'tree') "expected tree view, got $($tree.view)"
    $rootFromTree = $tree.comments | Where-Object { $_.id -eq $rootCommentID } | Select-Object -First 1
    $childFromTree = $tree.comments | Where-Object { $_.id -eq $childCommentID } | Select-Object -First 1
    Assert-True ($null -ne $rootFromTree) 'root comment missing from tree'
    Assert-True ($null -ne $childFromTree) 'child comment missing from tree'
    Assert-True ($rootFromTree.parent_id -eq $null) 'root parent_id should be null'
    Assert-True ($rootFromTree.depth -eq 0) "expected root depth 0, got $($rootFromTree.depth)"
    Assert-True ($childFromTree.parent_id -eq $rootCommentID) 'child parent_id mismatch'
    Assert-True ($childFromTree.depth -eq 1) "expected child depth 1, got $($childFromTree.depth)"
    Assert-True ($rootFromTree.attachments.Count -eq 1) 'root tree attachment missing'
    Assert-True ($childFromTree.attachments.Count -eq 1) 'child tree attachment missing'
    Assert-True ($rootFromTree.body_format -eq 'markdown' -and $childFromTree.body_format -eq 'markdown') 'tree body_format should be markdown'

    $rootIndex = [array]::IndexOf($tree.comments.id, $rootCommentID)
    $childIndex = [array]::IndexOf($tree.comments.id, $childCommentID)
    Assert-True ($rootIndex -ge 0 -and $childIndex -ge 0 -and $rootIndex -lt $childIndex) 'tree preorder is invalid: parent must appear before child'

    Assert-HttpErrorCode -StatusCode 400 -ErrorCode 'invalid_argument' -Request {
        Invoke-RestMethod -Uri "$baseURL/api/v1/posts/$postID/comments?view=nested" -Method Get -Headers $headers
    }

    [pscustomobject]@{
        status = 'passed'
        user = $username
        post_id = $postID
        post_attachment_id = $postAttachmentID
        root_comment_id = $rootCommentID
        root_attachment_id = $rootAttachmentID
        child_comment_id = $childCommentID
        child_attachment_id = $childAttachmentID
        tree_view = $tree.view
        storage_provider = 'local'
        base_url = $baseURL
    } | ConvertTo-Json -Compress
} finally {
    if ($job) {
        Stop-Job $job -ErrorAction SilentlyContinue
        Remove-Job $job -Force -ErrorAction SilentlyContinue
    }
    if ($smokeRoot.StartsWith((Join-Path $repo '.tmp'))) {
        Remove-Item -LiteralPath $smokeRoot -Recurse -Force -ErrorAction SilentlyContinue
    }
}
