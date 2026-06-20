param(
    [int] $Users = 1000,
    [int] $Communities = 50,
    [int] $Posts = 20000,
    [int] $Comments = 80000,
    [int] $PostVotes = 120000,
    [int] $PostSaves = 30000,
    [int] $Notifications = 12000,
    [int] $Reports = 3000,
    [int] $VUs = 50,
    [int] $DurationSeconds = 60,
    [int] $WarmupSeconds = 5,
    [int] $Port = 18080,
    [string] $Database = "cumt_nexus_loadtest",
    [string] $Include = "",
    [string] $Exclude = "",
    [double] $TargetRPS = 0,
    [switch] $SkipSeed,
    [switch] $KeepApi
)

$ErrorActionPreference = "Stop"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..\..")
$resultDir = Join-Path $repoRoot "docs\reports\loadtest"
$tmpDir = Join-Path $repoRoot ".tmp"
New-Item -ItemType Directory -Force -Path $resultDir | Out-Null
New-Item -ItemType Directory -Force -Path $tmpDir | Out-Null

function Set-LoadTestEnv {
    $env:APP_NAME = "cumt-nexus-api"
    $env:APP_ENV = "test"
    $env:APP_STARTUP_TIMEOUT = "20s"
    $env:POSTGRES_HOST = "localhost"
    $env:POSTGRES_PORT = "5432"
    $env:POSTGRES_USER = "postgres"
    $env:POSTGRES_PASSWORD = "postgres"
    $env:POSTGRES_DATABASE = $Database
    $env:POSTGRES_SSL_MODE = "disable"
    $env:POSTGRES_MAX_CONNS = "50"
    $env:POSTGRES_MAX_CONN_LIFETIME = "5m"
    $env:POSTGRES_MAX_CONN_IDLE_TIME = "2m"
    $env:HTTP_ADDR = ":$Port"
    $env:HTTP_READ_TIMEOUT = "5s"
    $env:HTTP_WRITE_TIMEOUT = "10s"
    $env:HTTP_SHUTDOWN_TIMEOUT = "15s"
    $env:HTTP_CORS_ALLOWED_ORIGINS = "http://localhost:3000,http://127.0.0.1:3000"
    $env:LOG_LEVEL = "warn"
    $env:LOG_FORMAT = "json"
    $env:GIN_MODE = "release"
    $env:AUTH_TOKEN_SECRET = "loadtest-auth-secret"
    $env:AUTH_ACCESS_TOKEN_TTL = "24h"
    $env:AUTH_EMAIL_ALLOWED_DOMAINS = "cumt.edu.cn,mail.cumt.edu.cn"
    $env:AUTH_EMAIL_CODE_TTL = "10m"
    $env:AUTH_EMAIL_CODE_RESEND_INTERVAL = "1m"
    $env:AUTH_EMAIL_CODE_MAX_ATTEMPTS = "5"
    $env:AUTH_EMAIL_CODE_DAILY_LIMIT = "10"
    $env:AUTH_EMAIL_CODE_IP_HOURLY_LIMIT = "30"
    $env:AUTH_EMAIL_CODE_LENGTH = "6"
    $env:MAIL_PROVIDER = "log"
    $env:SMTP_HOST = ""
    $env:SMTP_PORT = "587"
    $env:SMTP_USERNAME = ""
    $env:SMTP_PASSWORD = ""
    $env:SMTP_FROM = ""
    $env:SMTP_TLS_MODE = "starttls"
    $env:OBJECT_STORAGE_PROVIDER = "local"
    $env:OBJECT_STORAGE_ENDPOINT = ""
    $env:OBJECT_STORAGE_REGION = "auto"
    $env:OBJECT_STORAGE_BUCKET = ""
    $env:OBJECT_STORAGE_ACCESS_KEY_ID = ""
    $env:OBJECT_STORAGE_SECRET_ACCESS_KEY = ""
    $env:OBJECT_STORAGE_PUBLIC_BASE_URL = "http://localhost:$Port/uploads"
    $env:OBJECT_STORAGE_FORCE_PATH_STYLE = "true"
    $env:OBJECT_STORAGE_LOCAL_ROOT = "var/uploads-loadtest"
    $env:UPLOAD_IMAGE_MAX_BYTES = "5242880"
    $env:UPLOAD_IMAGE_MAX_COUNT_PER_POST = "9"
    $env:UPLOAD_IMAGE_MAX_COUNT_PER_COMMENT = "1"
}

function Invoke-Native {
    param(
        [Parameter(Mandatory = $true)]
        [scriptblock] $Command,
        [Parameter(Mandatory = $true)]
        [string] $Name
    )
    & $Command
    if ($LASTEXITCODE -ne 0) {
        throw "$Name failed with exit code $LASTEXITCODE"
    }
}

function Wait-HttpOk([string] $Url, [int] $TimeoutSeconds) {
    $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
    do {
        try {
            $response = Invoke-WebRequest -Uri $Url -UseBasicParsing -TimeoutSec 2
            if ($response.StatusCode -ge 200 -and $response.StatusCode -lt 300) {
                return
            }
        }
        catch {
            Start-Sleep -Milliseconds 500
        }
    } while ((Get-Date) -lt $deadline)
    throw "Timed out waiting for $Url"
}

function Wait-PostgresReady([int] $TimeoutSeconds) {
    $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
    do {
        docker exec cumt-nexus-postgres pg_isready -U postgres -d postgres *> $null
        if ($LASTEXITCODE -eq 0) {
            return
        }
        Start-Sleep -Milliseconds 500
    } while ((Get-Date) -lt $deadline)
    throw "Timed out waiting for PostgreSQL readiness"
}

function Stop-ProcessOnPort([int] $Port) {
    $lines = netstat -ano | Select-String ":$Port"
    foreach ($line in $lines) {
        $parts = ($line.ToString() -split "\s+") | Where-Object { $_ -ne "" }
        if ($parts.Length -lt 5) {
            continue
        }
        if ($parts[1] -notmatch ":$Port$" -or $parts[3] -ne "LISTENING") {
            continue
        }
        $pidValue = [int] $parts[4]
        try {
            $process = Get-Process -Id $pidValue -ErrorAction Stop
            if ($process.ProcessName -in @("api", "loadtest-api")) {
                Stop-Process -Id $pidValue -Force
            }
            else {
                throw "Port $Port is already used by process $pidValue ($($process.ProcessName)); refusing to stop a non-api process."
            }
        }
        catch {
            if ($_.Exception.Message -like "Port $Port is already used*") {
                throw
            }
        }
    }
}

Push-Location $repoRoot
try {
    Set-LoadTestEnv

    Invoke-Native -Name "docker compose up postgres" -Command { docker compose up -d postgres }
    Wait-PostgresReady 60
    $databaseExists = docker exec cumt-nexus-postgres psql -U postgres -d postgres -tc "SELECT 1 FROM pg_database WHERE datname = '$Database'" |
        Select-String "1" -Quiet
    if (-not $databaseExists) {
        Invoke-Native -Name "createdb $Database" -Command { docker exec cumt-nexus-postgres createdb -U postgres $Database }
    }

    Invoke-Native -Name "go run ./cmd/migrate up" -Command { go run ./cmd/migrate up }

    if (-not $SkipSeed) {
        Invoke-Native -Name "go run ./scripts/loadtest/cmd/seed" -Command { go run ./scripts/loadtest/cmd/seed `
            -users $Users `
            -communities $Communities `
            -posts $Posts `
            -comments $Comments `
            -post-votes $PostVotes `
            -post-saves $PostSaves `
            -notifications $Notifications `
            -reports $Reports `
            -reset | Tee-Object -FilePath (Join-Path $resultDir "seed-summary.json") }
    }

    Stop-ProcessOnPort $Port

    $apiStdoutLog = Join-Path $tmpDir "loadtest-api.stdout.log"
    $apiStderrLog = Join-Path $tmpDir "loadtest-api.stderr.log"
    $apiBinary = Join-Path $tmpDir "loadtest-api.exe"
    foreach ($apiLog in @($apiStdoutLog, $apiStderrLog)) {
        if (Test-Path $apiLog) {
            Remove-Item $apiLog -Force
        }
    }
    if (Test-Path $apiBinary) {
        Remove-Item $apiBinary -Force
    }
    Invoke-Native -Name "go build ./cmd/api" -Command { go build -o $apiBinary ./cmd/api }
    $api = Start-Process -FilePath $apiBinary -WorkingDirectory $repoRoot -NoNewWindow -RedirectStandardOutput $apiStdoutLog -RedirectStandardError $apiStderrLog -PassThru
    try {
        Wait-HttpOk "http://127.0.0.1:$Port/healthz" 60

        $stamp = Get-Date -Format "yyyyMMdd-HHmmss"
        $jsonPath = Join-Path $resultDir "loadtest-$stamp.json"
        $mdPath = Join-Path $resultDir "loadtest-$stamp.md"
        $runnerArgs = @(
            "run", "./scripts/loadtest/cmd/runner",
            "-base-url", "http://127.0.0.1:$Port",
            "-vus", $VUs,
            "-duration", "${DurationSeconds}s",
            "-warmup", "${WarmupSeconds}s",
            "-users", $Users,
            "-communities", $Communities,
            "-posts", $Posts,
            "-comments", $Comments,
            "-notifications", $Notifications,
            "-reports", $Reports
        )
        if (-not [string]::IsNullOrWhiteSpace($Include)) {
            $runnerArgs += @("-include", $Include)
        }
        if (-not [string]::IsNullOrWhiteSpace($Exclude)) {
            $runnerArgs += @("-exclude", $Exclude)
        }
        if ($TargetRPS -gt 0) {
            $runnerArgs += @("-target-rps", $TargetRPS)
        }
        $runnerArgs += @("-out-json", $jsonPath, "-out-md", $mdPath)
        Invoke-Native -Name "go run ./scripts/loadtest/cmd/runner" -Command { & go @runnerArgs }

        Write-Host "JSON report: $jsonPath"
        Write-Host "Markdown report: $mdPath"
    }
    finally {
        if (-not $KeepApi -and $api -and -not $api.HasExited) {
            Stop-Process -Id $api.Id -Force
        }
        if (-not $KeepApi) {
            Stop-ProcessOnPort $Port
        }
    }
}
finally {
    Pop-Location
}


