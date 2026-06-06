param(
    [int]$Port = 18082,
    [switch]$SkipMigration,
    [switch]$SkipWhenMissingCredentials
)

$ErrorActionPreference = 'Stop'

function Read-DotEnv {
    param([string]$Path)

    $values = @{}
    if (-not (Test-Path -LiteralPath $Path)) {
        return $values
    }

    foreach ($line in Get-Content -LiteralPath $Path) {
        $trimmed = $line.Trim()
        if ($trimmed -eq '' -or $trimmed.StartsWith('#')) {
            continue
        }
        $parts = $trimmed.Split('=', 2)
        if ($parts.Count -ne 2) {
            continue
        }
        $key = $parts[0].Trim()
        $value = $parts[1].Trim()
        if (($value.StartsWith('"') -and $value.EndsWith('"')) -or ($value.StartsWith("'") -and $value.EndsWith("'"))) {
            $value = $value.Substring(1, $value.Length - 2)
        }
        if ($key -ne '') {
            $values[$key] = $value
        }
    }
    return $values
}

function Resolve-ConfigValue {
    param(
        [string]$Key,
        [hashtable]$DotEnvValues,
        [string]$DefaultValue = ''
    )

    $value = [Environment]::GetEnvironmentVariable($Key)
    if (-not [string]::IsNullOrWhiteSpace($value)) {
        return $value.Trim()
    }
    if ($DotEnvValues.ContainsKey($Key) -and -not [string]::IsNullOrWhiteSpace($DotEnvValues[$Key])) {
        return [string]$DotEnvValues[$Key]
    }
    return $DefaultValue
}

function Assert-True {
    param(
        [bool]$Condition,
        [string]$Message
    )
    if (-not $Condition) {
        throw $Message
    }
}

function Assert-PublicImageReadable {
    param([string]$URL)

    $method = 'HEAD'
    try {
        $response = Invoke-WebRequest -Uri $URL -Method Head -MaximumRedirection 3 -TimeoutSec 15 -UseBasicParsing
    } catch {
        $method = 'GET'
        try {
            $response = Invoke-WebRequest -Uri $URL -Method Get -MaximumRedirection 3 -TimeoutSec 15 -UseBasicParsing
        } catch {
            throw "attachment public url is not readable: $URL error=$($_.Exception.Message)"
        }
    }

    Assert-True ($response.StatusCode -eq 200) "attachment public url returned HTTP $($response.StatusCode): $URL"
    $contentType = $response.Headers['Content-Type']
    if ($contentType -is [array]) {
        $contentType = $contentType[0]
    }
    $contentType = [string]$contentType
    Assert-True (-not [string]::IsNullOrWhiteSpace($contentType)) "attachment public url did not return Content-Type: $URL"
    Assert-True ($contentType.StartsWith('image/')) "attachment public url returned non-image Content-Type $contentType`: $URL"

    [pscustomobject]@{
        method = $method
        status_code = $response.StatusCode
        content_type = $contentType
    }
}

$repo = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$dotEnv = Read-DotEnv -Path (Join-Path $repo '.env')

$r2 = @{
    Endpoint = Resolve-ConfigValue -Key 'OBJECT_STORAGE_ENDPOINT' -DotEnvValues $dotEnv
    Region = Resolve-ConfigValue -Key 'OBJECT_STORAGE_REGION' -DotEnvValues $dotEnv -DefaultValue 'auto'
    Bucket = Resolve-ConfigValue -Key 'OBJECT_STORAGE_BUCKET' -DotEnvValues $dotEnv
    AccessKeyID = Resolve-ConfigValue -Key 'OBJECT_STORAGE_ACCESS_KEY_ID' -DotEnvValues $dotEnv
    SecretAccessKey = Resolve-ConfigValue -Key 'OBJECT_STORAGE_SECRET_ACCESS_KEY' -DotEnvValues $dotEnv
    PublicBaseURL = Resolve-ConfigValue -Key 'OBJECT_STORAGE_PUBLIC_BASE_URL' -DotEnvValues $dotEnv
    ForcePathStyle = Resolve-ConfigValue -Key 'OBJECT_STORAGE_FORCE_PATH_STYLE' -DotEnvValues $dotEnv -DefaultValue 'true'
}

$missing = @()
foreach ($entry in @(
    @{ Name = 'OBJECT_STORAGE_ENDPOINT'; Value = $r2.Endpoint },
    @{ Name = 'OBJECT_STORAGE_BUCKET'; Value = $r2.Bucket },
    @{ Name = 'OBJECT_STORAGE_ACCESS_KEY_ID'; Value = $r2.AccessKeyID },
    @{ Name = 'OBJECT_STORAGE_SECRET_ACCESS_KEY'; Value = $r2.SecretAccessKey },
    @{ Name = 'OBJECT_STORAGE_PUBLIC_BASE_URL'; Value = $r2.PublicBaseURL }
)) {
    if ([string]::IsNullOrWhiteSpace($entry.Value)) {
        $missing += $entry.Name
    }
}

if ($missing.Count -gt 0) {
    $result = [pscustomobject]@{
        status = 'skipped'
        reason = 'missing_r2_credentials'
        missing = $missing
    }
    if ($SkipWhenMissingCredentials) {
        $result | ConvertTo-Json -Compress
        exit 0
    }
    throw "missing R2 configuration: $($missing -join ', ')"
}

$smokeRoot = Join-Path $repo (Join-Path '.tmp' ('s15-r2-smoke-' + [guid]::NewGuid().ToString('N')))
$apiExe = Join-Path $smokeRoot 'api-smoke.exe'
$job = $null

try {
    New-Item -ItemType Directory -Force -Path $smokeRoot | Out-Null

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
        param($RepoPath, $ExePath, $ListenPort, $R2Config)

        Set-Location $RepoPath
        $env:APP_NAME = 'cumt-nexus-api-r2-smoke'
        $env:APP_ENV = 'local'
        $env:APP_STARTUP_TIMEOUT = '10s'
        $env:HTTP_ADDR = "127.0.0.1:$ListenPort"
        $env:AUTH_TOKEN_SECRET = 'SmokeAuthSecretForStage15R2'
        $env:OBJECT_STORAGE_PROVIDER = 'r2'
        $env:OBJECT_STORAGE_ENDPOINT = $R2Config.Endpoint
        $env:OBJECT_STORAGE_REGION = $R2Config.Region
        $env:OBJECT_STORAGE_BUCKET = $R2Config.Bucket
        $env:OBJECT_STORAGE_ACCESS_KEY_ID = $R2Config.AccessKeyID
        $env:OBJECT_STORAGE_SECRET_ACCESS_KEY = $R2Config.SecretAccessKey
        $env:OBJECT_STORAGE_PUBLIC_BASE_URL = $R2Config.PublicBaseURL
        $env:OBJECT_STORAGE_FORCE_PATH_STYLE = $R2Config.ForcePathStyle
        $env:UPLOAD_IMAGE_MAX_BYTES = '5242880'
        $env:UPLOAD_IMAGE_MAX_COUNT_PER_POST = '9'
        $env:UPLOAD_IMAGE_MAX_COUNT_PER_COMMENT = '1'

        & $ExePath
    } -ArgumentList $repo, $apiExe, $Port, ([pscustomobject]$r2)

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
    $username = "s15_r2_smoke_$suffix"
    $registerBody = @{ username = $username; password = 'password123' } | ConvertTo-Json -Compress
    $register = Invoke-RestMethod -Uri "$baseURL/api/v1/auth/register" -Method Post -ContentType 'application/json' -Body $registerBody
    $token = $register.access_token
    Assert-True (-not [string]::IsNullOrWhiteSpace($token)) 'register did not return access_token'
    $headers = @{ Authorization = "Bearer $token" }

    $pngPath = Join-Path $smokeRoot 'r2-smoke.png'
    [IO.File]::WriteAllBytes($pngPath, [Convert]::FromBase64String('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+/p9sAAAAASUVORK5CYII='))

    $uploadRaw = & curl.exe -sS -X POST "$baseURL/api/v1/uploads/images" -H "Authorization: Bearer $token" -F "file=@$pngPath" -F "alt_text=r2 smoke"
    if ($LASTEXITCODE -ne 0) {
        throw "curl upload failed: $uploadRaw"
    }
    $upload = $uploadRaw | ConvertFrom-Json
    $attachmentID = $upload.attachment.id
    Assert-True (-not [string]::IsNullOrWhiteSpace($attachmentID)) "upload missing attachment id: $uploadRaw"
    Assert-True ($upload.attachment.status -eq 'ready') "expected ready attachment, got $($upload.attachment.status)"
    Assert-True ($upload.attachment.mime_type -eq 'image/png') "expected image/png, got $($upload.attachment.mime_type)"

    $publicBase = $r2.PublicBaseURL.TrimEnd('/')
    Assert-True ($upload.attachment.url.StartsWith($publicBase + '/')) "attachment url does not use configured public base url"
    $publicURLCheck = Assert-PublicImageReadable -URL $upload.attachment.url

    $postBody = @{
        title = 'Stage 15 R2 smoke post'
        body = 'R2 smoke Markdown-like post body'
        attachment_ids = @($attachmentID)
    } | ConvertTo-Json -Compress
    $post = Invoke-RestMethod -Uri "$baseURL/api/v1/communities/public/posts" -Method Post -Headers $headers -ContentType 'application/json' -Body $postBody
    $postID = $post.post.id
    Assert-True (-not [string]::IsNullOrWhiteSpace($postID)) 'post create did not return id'
    Assert-True ($post.post.attachments.Count -eq 1) 'post create did not return bound attachment'
    Assert-True ($post.post.attachments[0].id -eq $attachmentID) 'post attachment id mismatch'
    Assert-True ($post.post.attachments[0].url.StartsWith($publicBase + '/')) 'post attachment url does not use configured public base url'

    [pscustomobject]@{
        status = 'passed'
        user = $username
        post_id = $postID
        attachment_id = $attachmentID
        attachment_url = $upload.attachment.url
        public_url_check = $publicURLCheck
        storage_provider = 'r2'
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
