param(
    [string]$DocPath = 'docs/contracts/configuration.md',
    [string]$EnvExamplePath = '.env.example'
)

$ErrorActionPreference = 'Stop'

$repo = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$loadPath = Join-Path $repo 'internal/platform/config/load.go'
$docFullPath = Join-Path $repo $DocPath
$envExampleFullPath = Join-Path $repo $EnvExamplePath

function Sorted {
    param([System.Collections.IEnumerable]$Values)
    return @($Values | Sort-Object)
}

if (-not (Test-Path -LiteralPath $loadPath)) {
    throw 'config loader not found'
}
if (-not (Test-Path -LiteralPath $docFullPath)) {
    throw "configuration doc not found: $DocPath"
}
if (-not (Test-Path -LiteralPath $envExampleFullPath)) {
    throw "env example not found: $EnvExamplePath"
}

$loadedKeys = [System.Collections.Generic.HashSet[string]]::new([StringComparer]::Ordinal)
$loadContent = Get-Content -Raw -Encoding UTF8 -LiteralPath $loadPath
$loadMatches = [regex]::Matches($loadContent, '(?:requiredString|stringDefault|stringListDefault|intDefault|boolDefault|durationDefault)\("([A-Z0-9_]+)"')
foreach ($match in $loadMatches) {
    [void]$loadedKeys.Add($match.Groups[1].Value)
}

$envExampleKeys = [System.Collections.Generic.HashSet[string]]::new([StringComparer]::Ordinal)
foreach ($line in Get-Content -Encoding UTF8 -LiteralPath $envExampleFullPath) {
    $trimmed = $line.Trim()
    if ($trimmed -eq '' -or $trimmed.StartsWith('#')) {
        continue
    }
    $match = [regex]::Match($trimmed, '^([A-Z0-9_]+)\s*=')
    if ($match.Success) {
        [void]$envExampleKeys.Add($match.Groups[1].Value)
    }
}

$docKeys = [System.Collections.Generic.HashSet[string]]::new([StringComparer]::Ordinal)
$docContent = Get-Content -Raw -Encoding UTF8 -LiteralPath $docFullPath
$docMatches = [regex]::Matches($docContent, '`([A-Z][A-Z0-9_]+)`')
foreach ($match in $docMatches) {
    [void]$docKeys.Add($match.Groups[1].Value)
}

$missingInEnvExample = @($loadedKeys | Where-Object { -not $envExampleKeys.Contains($_) })
$unknownInEnvExample = @($envExampleKeys | Where-Object { -not $loadedKeys.Contains($_) })
$missingInDoc = @($loadedKeys | Where-Object { -not $docKeys.Contains($_) })
$unknownInDoc = @($docKeys | Where-Object { -not $loadedKeys.Contains($_) })

if ($missingInEnvExample.Count -gt 0 -or $unknownInEnvExample.Count -gt 0 -or $missingInDoc.Count -gt 0 -or $unknownInDoc.Count -gt 0) {
    [pscustomobject]@{
        status = 'failed'
        loaded_key_count = $loadedKeys.Count
        missing_in_env_example = Sorted $missingInEnvExample
        unknown_in_env_example = Sorted $unknownInEnvExample
        missing_in_doc = Sorted $missingInDoc
        unknown_in_doc = Sorted $unknownInDoc
    } | ConvertTo-Json -Depth 4
    throw 'configuration contract is out of sync'
}

[pscustomobject]@{
    status = 'passed'
    loaded_key_count = $loadedKeys.Count
    doc = $DocPath
    env_example = $EnvExamplePath
    keys = Sorted $loadedKeys
} | ConvertTo-Json -Depth 4
