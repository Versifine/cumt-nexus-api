param(
    [switch]$SkipHttpSmoke,
    [switch]$SkipContractDocChecks,
    [ValidateSet('SkipWhenMissing', 'Require', 'Skip')]
    [string]$R2Mode = 'SkipWhenMissing',
    [int]$Stage13Port = 18130,
    [int]$Stage14Port = 18131,
    [int]$Stage15Port = 18132
)

$ErrorActionPreference = 'Stop'

$repo = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$results = [System.Collections.Generic.List[object]]::new()
$powerShellExecutable = $null
foreach ($candidate in @('pwsh', 'powershell')) {
    $command = Get-Command $candidate -ErrorAction SilentlyContinue
    if ($command) {
        $powerShellExecutable = $command.Source
        break
    }
}
if (-not $powerShellExecutable) {
    throw 'PowerShell executable not found: expected pwsh or powershell'
}

function Invoke-Native {
    param(
        [string]$File,
        [string[]]$Arguments
    )

    & $File @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw "$File $($Arguments -join ' ') failed with exit code $LASTEXITCODE"
    }
}

function Invoke-Step {
    param(
        [string]$Name,
        [scriptblock]$Command
    )

    $startedAt = Get-Date
    Write-Host "==> $Name"
    try {
        & $Command
        $duration = [math]::Round(((Get-Date) - $startedAt).TotalSeconds, 3)
        $script:results.Add([pscustomobject]@{
            name = $Name
            status = 'passed'
            duration_seconds = $duration
        }) | Out-Null
        Write-Host "<== $Name passed"
    } catch {
        $duration = [math]::Round(((Get-Date) - $startedAt).TotalSeconds, 3)
        $script:results.Add([pscustomobject]@{
            name = $Name
            status = 'failed'
            duration_seconds = $duration
            error = $_.Exception.Message
        }) | Out-Null
        throw
    }
}

function Add-SkippedStep {
    param(
        [string]$Name,
        [string]$Reason
    )

    Write-Host "==> $Name"
    $script:results.Add([pscustomobject]@{
        name = $Name
        status = 'skipped'
        duration_seconds = 0
        reason = $Reason
    }) | Out-Null
    Write-Host "<== $Name skipped: $Reason"
}

Push-Location $repo
try {
    if ($SkipContractDocChecks) {
        Add-SkippedStep -Name 'contract docs inventory' -Reason 'SkipContractDocChecks'
    } else {
        Invoke-Step -Name 'api contract route/auth/query inventory' -Command {
            Invoke-Native -File $powerShellExecutable -Arguments @(
                '-NoProfile',
                '-ExecutionPolicy',
                'Bypass',
                '-File',
                (Join-Path $repo 'scripts/verify-api-contract-doc.ps1')
            )
        }

        Invoke-Step -Name 'api schema fields/routes/required inventory' -Command {
            Invoke-Native -File $powerShellExecutable -Arguments @(
                '-NoProfile',
                '-ExecutionPolicy',
                'Bypass',
                '-File',
                (Join-Path $repo 'scripts/verify-api-schema-doc.ps1')
            )
        }

        Invoke-Step -Name 'http error contract inventory' -Command {
            Invoke-Native -File $powerShellExecutable -Arguments @(
                '-NoProfile',
                '-ExecutionPolicy',
                'Bypass',
                '-File',
                (Join-Path $repo 'scripts/verify-http-error-contract-doc.ps1')
            )
        }

        Invoke-Step -Name 'configuration contract inventory' -Command {
            Invoke-Native -File $powerShellExecutable -Arguments @(
                '-NoProfile',
                '-ExecutionPolicy',
                'Bypass',
                '-File',
                (Join-Path $repo 'scripts/verify-config-contract-doc.ps1')
            )
        }

        Invoke-Step -Name 'configuration semantic contract' -Command {
            Invoke-Native -File $powerShellExecutable -Arguments @(
                '-NoProfile',
                '-ExecutionPolicy',
                'Bypass',
                '-File',
                (Join-Path $repo 'scripts/verify-config-semantics-doc.ps1')
            )
        }

        Invoke-Step -Name 'migration contract inventory' -Command {
            Invoke-Native -File $powerShellExecutable -Arguments @(
                '-NoProfile',
                '-ExecutionPolicy',
                'Bypass',
                '-File',
                (Join-Path $repo 'scripts/verify-migration-contract.ps1')
            )
        }
    }

    Invoke-Step -Name 'go run ./cmd/migrate up' -Command {
        Invoke-Native -File 'go' -Arguments @('run', './cmd/migrate', 'up')
    }

    Invoke-Step -Name 'go test ./...' -Command {
        Invoke-Native -File 'go' -Arguments @('test', './...')
    }

    Invoke-Step -Name 'go build -buildvcs=false ./...' -Command {
        Invoke-Native -File 'go' -Arguments @('build', '-buildvcs=false', './...')
    }

    if ($SkipHttpSmoke) {
        Add-SkippedStep -Name 'stage 13 content smoke' -Reason 'SkipHttpSmoke'
        Add-SkippedStep -Name 'stage 14 content lifecycle smoke' -Reason 'SkipHttpSmoke'
    } else {
        Invoke-Step -Name 'stage 13 content smoke' -Command {
            Invoke-Native -File $powerShellExecutable -Arguments @(
                '-NoProfile',
                '-ExecutionPolicy',
                'Bypass',
                '-File',
                (Join-Path $repo 'scripts/smoke-stage-13-content-system.ps1'),
                '-Port',
                [string]$Stage13Port,
                '-SkipMigration'
            )
        }

        Invoke-Step -Name 'stage 14 content lifecycle smoke' -Command {
            Invoke-Native -File $powerShellExecutable -Arguments @(
                '-NoProfile',
                '-ExecutionPolicy',
                'Bypass',
                '-File',
                (Join-Path $repo 'scripts/smoke-stage-14-content-lifecycle.ps1'),
                '-Port',
                [string]$Stage14Port,
                '-SkipMigration'
            )
        }
    }

    switch ($R2Mode) {
        'Skip' {
            Add-SkippedStep -Name 'stage 15 R2 smoke' -Reason 'R2Mode=Skip'
        }
        'Require' {
            Invoke-Step -Name 'stage 15 R2 smoke' -Command {
                Invoke-Native -File $powerShellExecutable -Arguments @(
                    '-NoProfile',
                    '-ExecutionPolicy',
                    'Bypass',
                    '-File',
                    (Join-Path $repo 'scripts/smoke-stage-15-r2-upload.ps1'),
                    '-Port',
                    [string]$Stage15Port,
                    '-SkipMigration'
                )
            }
        }
        'SkipWhenMissing' {
            Invoke-Step -Name 'stage 15 R2 smoke or credential gate' -Command {
                Invoke-Native -File $powerShellExecutable -Arguments @(
                    '-NoProfile',
                    '-ExecutionPolicy',
                    'Bypass',
                    '-File',
                    (Join-Path $repo 'scripts/smoke-stage-15-r2-upload.ps1'),
                    '-Port',
                    [string]$Stage15Port,
                    '-SkipMigration',
                    '-SkipWhenMissingCredentials'
                )
            }
        }
    }

    [pscustomobject]@{
        status = 'passed'
        r2_mode = $R2Mode
        http_smoke_skipped = [bool]$SkipHttpSmoke
        contract_doc_checks_skipped = [bool]$SkipContractDocChecks
        steps = $results
    } | ConvertTo-Json -Depth 5
} finally {
    Pop-Location
}
