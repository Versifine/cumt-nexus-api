param(
    [string]$DocPath = 'docs/internal/architecture/http-api-schema.md'
)

$ErrorActionPreference = 'Stop'

$repo = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$docFullPath = Join-Path $repo $DocPath

function New-SchemaKey {
    param(
        [string]$Package,
        [string]$Type
    )
    return "$Package.$Type"
}

function Get-StructSchemasFromFile {
    param([string]$Path)

    $content = Get-Content -Raw -Encoding UTF8 -LiteralPath $Path
    $packageMatch = [regex]::Match($content, '(?m)^package\s+(\w+)')
    if (-not $packageMatch.Success) {
        return @()
    }
    $packageName = $packageMatch.Groups[1].Value
    $schemas = @()
    $structMatches = [regex]::Matches($content, '(?s)type\s+(\w+)\s+struct\s*\{(.*?)\n\}')
    foreach ($structMatch in $structMatches) {
        $typeName = $structMatch.Groups[1].Value
        $body = $structMatch.Groups[2].Value
        $fieldMatches = [regex]::Matches($body, 'json:"([^"]+)"')
        $fields = @()
        foreach ($fieldMatch in $fieldMatches) {
            $tagValue = $fieldMatch.Groups[1].Value
            $fieldName = $tagValue.Split(',', 2)[0]
            if ($fieldName -ne '' -and $fieldName -ne '-') {
                $fields += $fieldName
            }
        }
        if ($fields.Count -gt 0) {
            $schemas += [pscustomobject]@{
                Package = $packageName
                Type = $typeName
                Fields = $fields
                Source = (Resolve-Path -LiteralPath $Path).Path.Substring($repo.Length + 1).Replace('\', '/')
            }
        }
    }
    return $schemas
}

$deliveryFiles = Get-ChildItem -LiteralPath (Join-Path $repo 'internal') -Recurse -Filter '*.go' |
    Where-Object {
        $_.FullName -match '\\delivery\\.*http\\' -and
        $_.Name -notlike '*_test.go'
    }

$actualSchemas = @{}
foreach ($file in $deliveryFiles) {
    foreach ($schema in Get-StructSchemasFromFile -Path $file.FullName) {
        $key = New-SchemaKey -Package $schema.Package -Type $schema.Type
        if ($actualSchemas.ContainsKey($key)) {
            throw "duplicate handler schema key: $key"
        }
        $actualSchemas[$key] = $schema
    }
}

if (-not (Test-Path -LiteralPath $docFullPath)) {
    throw "API schema doc not found: $DocPath"
}

$documentedSchemas = @{}
foreach ($line in Get-Content -Encoding UTF8 -LiteralPath $docFullPath) {
    $match = [regex]::Match($line, '^\|\s*`([^`]+)`\s*\|\s*`([^`]+)`\s*\|\s*((?:`[^`]+`(?:,\s*)?)+)\s*\|')
    if (-not $match.Success) {
        continue
    }
    $packageName = $match.Groups[1].Value
    $typeName = $match.Groups[2].Value
    $fieldText = $match.Groups[3].Value
    $fields = @()
    foreach ($fieldMatch in [regex]::Matches($fieldText, '`([^`]+)`')) {
        $fields += $fieldMatch.Groups[1].Value
    }
    $key = New-SchemaKey -Package $packageName -Type $typeName
    if ($documentedSchemas.ContainsKey($key)) {
        throw "duplicate documented schema key: $key"
    }
    $documentedSchemas[$key] = $fields
}

$actualKeys = @($actualSchemas.Keys | Sort-Object)
$documentedKeys = @($documentedSchemas.Keys | Sort-Object)
$missingInDoc = @($actualKeys | Where-Object { -not $documentedSchemas.ContainsKey($_) })
$staleInDoc = @($documentedKeys | Where-Object { -not $actualSchemas.ContainsKey($_) })
$fieldMismatches = @()

foreach ($key in $actualKeys) {
    if (-not $documentedSchemas.ContainsKey($key)) {
        continue
    }
    $actualFields = @($actualSchemas[$key].Fields)
    $documentedFields = @($documentedSchemas[$key])
    if (($actualFields -join ',') -ne ($documentedFields -join ',')) {
        $fieldMismatches += [pscustomobject]@{
            schema = $key
            source = $actualSchemas[$key].Source
            actual = $actualFields
            documented = $documentedFields
        }
    }
}

if ($missingInDoc.Count -gt 0 -or $staleInDoc.Count -gt 0 -or $fieldMismatches.Count -gt 0) {
    [pscustomobject]@{
        status = 'failed'
        missing_in_doc = $missingInDoc
        stale_in_doc = $staleInDoc
        field_mismatches = $fieldMismatches
    } | ConvertTo-Json -Depth 6
    throw 'API schema doc is out of sync'
}

[pscustomobject]@{
    status = 'passed'
    schema_count = $actualSchemas.Count
    doc = $DocPath
    schemas = $actualKeys
} | ConvertTo-Json -Depth 4
