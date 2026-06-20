param(
    [Parameter(Mandatory = $true)]
    [string] $Endpoint,
    [string] $RpsSteps = "5,10,15,20,25,30",
    [int] $StepDurationSeconds = 60,
    [int] $WarmupSeconds = 10,
    [int] $VUs = 80,
    [int] $Users = 1000,
    [int] $Communities = 50,
    [int] $Posts = 20000,
    [int] $Comments = 80000,
    [int] $PostVotes = 120000,
    [int] $PostSaves = 30000,
    [int] $Notifications = 12000,
    [int] $Reports = 3000,
    [int] $Port = 18080,
    [string] $Database = "cumt_nexus_loadtest",
    [double] $P95ThresholdMs = 2000,
    [double] $ErrorRateThreshold = 0.01,
    [double] $ActualRpsRatioThreshold = 0.95,
    [switch] $SkipSeed
)

$ErrorActionPreference = "Stop"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..\..")
$resultDir = Join-Path $repoRoot "docs\reports\loadtest"
$tmpDir = Join-Path $repoRoot ".tmp"
$runLocalScript = Join-Path $PSScriptRoot "run-local-loadtest.ps1"
New-Item -ItemType Directory -Force -Path $resultDir | Out-Null
New-Item -ItemType Directory -Force -Path $tmpDir | Out-Null

$rpsStepValues = $RpsSteps -split "[,;\s]+" |
    Where-Object { -not [string]::IsNullOrWhiteSpace($_) } |
    ForEach-Object { [double] $_ }
if ($rpsStepValues.Count -eq 0) {
    throw "RpsSteps must contain at least one numeric target RPS"
}

function Invoke-LoadStep {
    param(
        [double] $TargetRPS,
        [bool] $ShouldSkipSeed
    )

    $args = @(
        "-File", $runLocalScript,
        "-Users", $Users,
        "-Communities", $Communities,
        "-Posts", $Posts,
        "-Comments", $Comments,
        "-PostVotes", $PostVotes,
        "-PostSaves", $PostSaves,
        "-Notifications", $Notifications,
        "-Reports", $Reports,
        "-VUs", $VUs,
        "-DurationSeconds", $StepDurationSeconds,
        "-WarmupSeconds", $WarmupSeconds,
        "-Port", $Port,
        "-Database", $Database,
        "-Include", $Endpoint,
        "-TargetRPS", $TargetRPS
    )
    if ($ShouldSkipSeed) {
        $args += "-SkipSeed"
    }

    $logPrefix = Join-Path $tmpDir ("fixed-rps-step-{0}-{1}" -f ($Endpoint -replace "[^a-zA-Z0-9_-]", "_"), $TargetRPS)
    $stdoutLog = "$logPrefix.stdout.log"
    $stderrLog = "$logPrefix.stderr.log"
    $process = Start-Process -FilePath "powershell" -ArgumentList (@("-NoProfile", "-ExecutionPolicy", "Bypass") + $args) -NoNewWindow -Wait -PassThru -RedirectStandardOutput $stdoutLog -RedirectStandardError $stderrLog
    $exitCode = $process.ExitCode
    $output = @()
    if (Test-Path $stdoutLog) {
        $output += Get-Content -Path $stdoutLog
    }
    if (Test-Path $stderrLog) {
        $output += Get-Content -Path $stderrLog
    }
    if ($exitCode -ne 0) {
        $output | ForEach-Object { Write-Host $_ }
        throw "fixed RPS step failed for endpoint=$Endpoint target_rps=$TargetRPS"
    }

    $jsonLine = $output | Where-Object { $_ -match "^JSON report:" } | Select-Object -Last 1
    if (-not $jsonLine) {
        $output | ForEach-Object { Write-Host $_ }
        throw "runner did not print JSON report path for endpoint=$Endpoint target_rps=$TargetRPS"
    }
    return (($jsonLine -replace "^JSON report:\s*", "").Trim())
}

function Format-Percent([double] $Value) {
    return ("{0:N2}%" -f ($Value * 100))
}

$startedAt = Get-Date
$rows = @()
$seedAlreadyHandled = $SkipSeed.IsPresent
foreach ($rps in $rpsStepValues) {
    Write-Host "Running fixed RPS step: endpoint=$Endpoint target_rps=$rps"
    $jsonPath = Invoke-LoadStep -TargetRPS $rps -ShouldSkipSeed $seedAlreadyHandled
    $seedAlreadyHandled = $true

    $report = Get-Content -Raw -Path $jsonPath | ConvertFrom-Json
    $endpointMetrics = $report.endpoints.$Endpoint
    if (-not $endpointMetrics) {
        throw "report $jsonPath does not contain endpoint $Endpoint"
    }
    $healthy = (
        [double] $endpointMetrics.error_rate -le $ErrorRateThreshold -and
        [double] $endpointMetrics.p95_ms -le $P95ThresholdMs -and
        [double] $endpointMetrics.rps -ge ($rps * $ActualRpsRatioThreshold)
    )
    $rows += [pscustomobject]@{
        endpoint = $Endpoint
        target_rps = [double] $rps
        actual_rps = [double] $endpointMetrics.rps
        requests = [int] $endpointMetrics.requests
        error_rate = [double] $endpointMetrics.error_rate
        p50_ms = [double] $endpointMetrics.p50_ms
        p95_ms = [double] $endpointMetrics.p95_ms
        p99_ms = [double] $endpointMetrics.p99_ms
        max_ms = [double] $endpointMetrics.max_ms
        healthy = [bool] $healthy
        json_report = $jsonPath
        markdown_report = [System.IO.Path]::ChangeExtension($jsonPath, ".md")
    }
}

$firstBad = $rows | Where-Object { -not $_.healthy } | Select-Object -First 1
$lastHealthy = $rows | Where-Object { $_.healthy } | Select-Object -Last 1
$finishedAt = Get-Date
$stamp = Get-Date -Format "yyyyMMdd-HHmmss"
$safeEndpoint = $Endpoint -replace "[^a-zA-Z0-9_-]", "_"
$summaryJsonPath = Join-Path $resultDir "fixed-rps-ladder-$safeEndpoint-$stamp.json"
$summaryMdPath = Join-Path $resultDir "fixed-rps-ladder-$safeEndpoint-$stamp.md"

$summary = [pscustomobject]@{
    endpoint = $Endpoint
    started_at = $startedAt.ToUniversalTime().ToString("o")
    finished_at = $finishedAt.ToUniversalTime().ToString("o")
    step_duration_seconds = $StepDurationSeconds
    warmup_seconds = $WarmupSeconds
    vus = $VUs
    thresholds = [pscustomobject]@{
        p95_ms = $P95ThresholdMs
        error_rate = $ErrorRateThreshold
        actual_rps_ratio = $ActualRpsRatioThreshold
    }
    last_healthy = $lastHealthy
    first_bad = $firstBad
    steps = $rows
}
$summary | ConvertTo-Json -Depth 8 | Set-Content -Encoding UTF8 -Path $summaryJsonPath

$md = New-Object System.Text.StringBuilder
[void] $md.AppendLine("# Fixed RPS Ladder Report")
[void] $md.AppendLine()
[void] $md.AppendLine("- Endpoint: ``$Endpoint``")
[void] $md.AppendLine("- Step duration: ``${StepDurationSeconds}s`` measured, ``${WarmupSeconds}s`` warmup")
[void] $md.AppendLine("- VUs: ``$VUs``")
[void] $md.AppendLine("- Healthy rule: error rate <= ``$(Format-Percent $ErrorRateThreshold)``, p95 <= ``$P95ThresholdMs ms``, actual RPS >= ``$([math]::Round($ActualRpsRatioThreshold * 100, 2))%`` of target")
if ($lastHealthy) {
    [void] $md.AppendLine("- Last healthy step: target ``$($lastHealthy.target_rps) req/s`` with p95 ``$([math]::Round($lastHealthy.p95_ms, 2)) ms``")
}
if ($firstBad) {
    [void] $md.AppendLine("- First failing step: target ``$($firstBad.target_rps) req/s`` with p95 ``$([math]::Round($firstBad.p95_ms, 2)) ms`` and error rate ``$(Format-Percent $firstBad.error_rate)``")
}
[void] $md.AppendLine()
[void] $md.AppendLine("| Target RPS | Actual RPS | Requests | Error Rate | p50 | p95 | p99 | Max | Healthy |")
[void] $md.AppendLine("|---:|---:|---:|---:|---:|---:|---:|---:|---|")
foreach ($row in $rows) {
    [void] $md.AppendLine((
        "| {0:N2} | {1:N2} | {2} | {3} | {4:N2} ms | {5:N2} ms | {6:N2} ms | {7:N2} ms | {8} |" -f
        $row.target_rps,
        $row.actual_rps,
        $row.requests,
        (Format-Percent $row.error_rate),
        $row.p50_ms,
        $row.p95_ms,
        $row.p99_ms,
        $row.max_ms,
        $row.healthy
    ))
}
[void] $md.AppendLine()
[void] $md.AppendLine("## Raw Reports")
[void] $md.AppendLine()
foreach ($row in $rows) {
    [void] $md.AppendLine("- Target ``$($row.target_rps) req/s``: ``$($row.markdown_report)``")
}
$md.ToString() | Set-Content -Encoding UTF8 -Path $summaryMdPath

Write-Host "Ladder JSON report: $summaryJsonPath"
Write-Host "Ladder Markdown report: $summaryMdPath"
