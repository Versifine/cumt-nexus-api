param(
    [int]$Port = 18081,
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
$smokeRoot = Join-Path $repo (Join-Path '.tmp' ('s14-smoke-' + [guid]::NewGuid().ToString('N')))
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
        $env:AUTH_TOKEN_SECRET = 'SmokeAuthSecretForStage14'
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
    $authorName = "s14_author_$suffix"
    $intruderName = "s14_intruder_$suffix"

    $authorRegisterBody = @{ username = $authorName; password = 'password123' } | ConvertTo-Json -Compress
    $author = Invoke-RestMethod -Uri "$baseURL/api/v1/auth/register" -Method Post -ContentType 'application/json' -Body $authorRegisterBody
    $authorToken = $author.access_token
    Assert-True (-not [string]::IsNullOrWhiteSpace($authorToken)) 'author register did not return access_token'
    $authorHeaders = @{ Authorization = "Bearer $authorToken" }

    $intruderRegisterBody = @{ username = $intruderName; password = 'password123' } | ConvertTo-Json -Compress
    $intruder = Invoke-RestMethod -Uri "$baseURL/api/v1/auth/register" -Method Post -ContentType 'application/json' -Body $intruderRegisterBody
    $intruderToken = $intruder.access_token
    Assert-True (-not [string]::IsNullOrWhiteSpace($intruderToken)) 'intruder register did not return access_token'
    $intruderHeaders = @{ Authorization = "Bearer $intruderToken" }

    $postBody = @{
        title = 'Stage 14 smoke post'
        body = 'Original Markdown-like post body'
    } | ConvertTo-Json -Compress
    $post = Invoke-RestMethod -Uri "$baseURL/api/v1/communities/public/posts" -Method Post -Headers $authorHeaders -ContentType 'application/json' -Body $postBody
    $postID = $post.post.id
    Assert-True (-not [string]::IsNullOrWhiteSpace($postID)) 'post create did not return id'

    $commentBody = @{
        body = 'Original Markdown-like comment body'
    } | ConvertTo-Json -Compress
    $comment = Invoke-RestMethod -Uri "$baseURL/api/v1/posts/$postID/comments" -Method Post -Headers $authorHeaders -ContentType 'application/json' -Body $commentBody
    $commentID = $comment.comment.id
    Assert-True (-not [string]::IsNullOrWhiteSpace($commentID)) 'comment create did not return id'

    Assert-HttpErrorCode -StatusCode 403 -ErrorCode 'forbidden' -Request {
        $body = @{ title = 'Intruder title'; body = 'Intruder body' } | ConvertTo-Json -Compress
        Invoke-RestMethod -Uri "$baseURL/api/v1/posts/$postID" -Method Patch -Headers $intruderHeaders -ContentType 'application/json' -Body $body
    }

    $updatedPostBody = @{
        title = 'Stage 14 smoke post updated'
        body = 'Updated Markdown-like post body'
    } | ConvertTo-Json -Compress
    $updatedPost = Invoke-RestMethod -Uri "$baseURL/api/v1/posts/$postID" -Method Patch -Headers $authorHeaders -ContentType 'application/json' -Body $updatedPostBody
    Assert-True ($updatedPost.post.title -eq 'Stage 14 smoke post updated') "post title was not updated: $($updatedPost.post.title)"
    Assert-True ($updatedPost.post.body -eq 'Updated Markdown-like post body') "post body was not updated: $($updatedPost.post.body)"
    Assert-True ($updatedPost.post.body_format -eq 'markdown') "post body_format should remain markdown, got $($updatedPost.post.body_format)"

    $postDetail = Invoke-RestMethod -Uri "$baseURL/api/v1/posts/$postID" -Method Get -Headers $authorHeaders
    Assert-True ($postDetail.post.title -eq 'Stage 14 smoke post updated') 'post detail did not reflect update'

    Assert-HttpErrorCode -StatusCode 403 -ErrorCode 'forbidden' -Request {
        $body = @{ body = 'Intruder comment body' } | ConvertTo-Json -Compress
        Invoke-RestMethod -Uri "$baseURL/api/v1/comments/$commentID" -Method Patch -Headers $intruderHeaders -ContentType 'application/json' -Body $body
    }

    $updatedCommentBody = @{ body = 'Updated Markdown-like comment body' } | ConvertTo-Json -Compress
    $updatedComment = Invoke-RestMethod -Uri "$baseURL/api/v1/comments/$commentID" -Method Patch -Headers $authorHeaders -ContentType 'application/json' -Body $updatedCommentBody
    Assert-True ($updatedComment.comment.body -eq 'Updated Markdown-like comment body') "comment body was not updated: $($updatedComment.comment.body)"
    Assert-True ($updatedComment.comment.body_format -eq 'markdown') "comment body_format should remain markdown, got $($updatedComment.comment.body_format)"

    $tree = Invoke-RestMethod -Uri "$baseURL/api/v1/posts/$postID/comments?view=tree&sort=new&limit=20&offset=0&max_depth=6" -Method Get -Headers $authorHeaders
    $commentFromTree = $tree.comments | Where-Object { $_.id -eq $commentID } | Select-Object -First 1
    Assert-True ($null -ne $commentFromTree) 'updated comment missing from tree'
    Assert-True ($commentFromTree.body -eq 'Updated Markdown-like comment body') 'tree did not reflect comment update'

    Assert-HttpErrorCode -StatusCode 403 -ErrorCode 'forbidden' -Request {
        Invoke-RestMethod -Uri "$baseURL/api/v1/comments/$commentID" -Method Delete -Headers $intruderHeaders
    }

    Invoke-RestMethod -Uri "$baseURL/api/v1/comments/$commentID" -Method Delete -Headers $authorHeaders | Out-Null
    $afterCommentDelete = Invoke-RestMethod -Uri "$baseURL/api/v1/posts/$postID/comments?view=tree&sort=new&limit=20&offset=0&max_depth=6" -Method Get -Headers $authorHeaders
    $deletedComment = $afterCommentDelete.comments | Where-Object { $_.id -eq $commentID } | Select-Object -First 1
    Assert-True ($null -eq $deletedComment) 'deleted comment should not appear in tree'

    Assert-HttpErrorCode -StatusCode 403 -ErrorCode 'forbidden' -Request {
        Invoke-RestMethod -Uri "$baseURL/api/v1/posts/$postID" -Method Delete -Headers $intruderHeaders
    }

    Invoke-RestMethod -Uri "$baseURL/api/v1/posts/$postID" -Method Delete -Headers $authorHeaders | Out-Null
    Assert-HttpErrorCode -StatusCode 404 -ErrorCode 'not_found' -Request {
        Invoke-RestMethod -Uri "$baseURL/api/v1/posts/$postID" -Method Get -Headers $authorHeaders
    }

    [pscustomobject]@{
        status = 'passed'
        author = $authorName
        intruder = $intruderName
        post_id = $postID
        comment_id = $commentID
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
