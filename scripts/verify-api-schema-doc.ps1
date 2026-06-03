param(
    [string]$DocPath = 'docs/internal/architecture/http-api-schema.md',
    [string]$RouteDocPath = 'docs/internal/architecture/http-api-contract.md'
)

$ErrorActionPreference = 'Stop'

$repo = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$docFullPath = Join-Path $repo $DocPath
$routeDocFullPath = Join-Path $repo $RouteDocPath

function New-SchemaKey {
    param(
        [string]$Package,
        [string]$Type
    )
    return "$Package.$Type"
}

function New-RouteKey {
    param(
        [string]$Method,
        [string]$Path
    )
    return "$($Method.ToUpperInvariant()) $Path"
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

function Read-RouteKeysFromDoc {
    param([string]$Path)

    $routes = [System.Collections.Generic.HashSet[string]]::new([StringComparer]::Ordinal)
    foreach ($line in Get-Content -Encoding UTF8 -LiteralPath $Path) {
        $match = [regex]::Match($line, '^\|\s*(GET|POST|PUT|PATCH|DELETE)\s*\|\s*([^|]+?)\s*\|')
        if ($match.Success) {
            [void]$routes.Add((New-RouteKey -Method $match.Groups[1].Value -Path $match.Groups[2].Value.Trim()))
        }
    }
    return $routes
}

function Read-SchemaRouteMappingsFromDoc {
    param([string]$Path)

    $mappings = @{}
    foreach ($line in Get-Content -Encoding UTF8 -LiteralPath $Path) {
        $match = [regex]::Match($line, '^\|\s*(GET|POST|PUT|PATCH|DELETE)\s*\|\s*([^|]+?)\s*\|\s*([^|]+?)\s*\|\s*([^|]+?)\s*\|\s*(\d+)\s*\|')
        if (-not $match.Success) {
            continue
        }
        $key = New-RouteKey -Method $match.Groups[1].Value -Path $match.Groups[2].Value.Trim()
        if ($mappings.ContainsKey($key)) {
            throw "duplicate API schema route mapping: $key"
        }
        $mappings[$key] = [pscustomobject]@{
            Method = $match.Groups[1].Value
            Path = $match.Groups[2].Value.Trim()
            Request = $match.Groups[3].Value.Trim()
            Success = $match.Groups[4].Value.Trim()
            Status = [int]$match.Groups[5].Value
        }
    }
    return $mappings
}

function Get-SchemaRefs {
    param([string]$Cell)

    $refs = @()
    foreach ($match in [regex]::Matches($Cell, '`([^`]+)`')) {
        $value = $match.Groups[1].Value
        if ([regex]::IsMatch($value, '^\w+http\.\w+$')) {
            $refs += $value
        }
    }
    return $refs
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
if (-not (Test-Path -LiteralPath $routeDocFullPath)) {
    throw "API route contract doc not found: $RouteDocPath"
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

$contractRouteKeys = Read-RouteKeysFromDoc -Path $routeDocFullPath
$schemaRouteMappings = Read-SchemaRouteMappingsFromDoc -Path $docFullPath
$schemaRouteKeys = [System.Collections.Generic.HashSet[string]]::new([StringComparer]::Ordinal)
foreach ($key in $schemaRouteMappings.Keys) {
    [void]$schemaRouteKeys.Add($key)
}

$missingRouteMappings = @($contractRouteKeys | Where-Object { -not $schemaRouteKeys.Contains($_) } | Sort-Object)
$staleRouteMappings = @($schemaRouteKeys | Where-Object { -not $contractRouteKeys.Contains($_) } | Sort-Object)
$invalidSchemaRefs = @()
$invalidStatuses = @()

foreach ($routeKey in ($schemaRouteMappings.Keys | Sort-Object)) {
    $mapping = $schemaRouteMappings[$routeKey]
    $schemaRefs = @()
    $schemaRefs += @(Get-SchemaRefs -Cell $mapping.Request)
    $schemaRefs += @(Get-SchemaRefs -Cell $mapping.Success)
    foreach ($schemaRef in $schemaRefs) {
        if ([string]::IsNullOrWhiteSpace($schemaRef)) {
            continue
        }
        if (-not $documentedSchemas.ContainsKey($schemaRef) -or -not $actualSchemas.ContainsKey($schemaRef)) {
            $invalidSchemaRefs += [pscustomobject]@{
                route = $routeKey
                schema = $schemaRef
            }
        }
    }
    if ($mapping.Status -notin @(200, 201, 204)) {
        $invalidStatuses += [pscustomobject]@{
            route = $routeKey
            status = $mapping.Status
        }
    }
    if ($mapping.Status -eq 204 -and $mapping.Success -ne 'none') {
        $invalidStatuses += [pscustomobject]@{
            route = $routeKey
            status = $mapping.Status
            reason = '204 success must be none'
        }
    }
}

if (
    $missingInDoc.Count -gt 0 -or
    $staleInDoc.Count -gt 0 -or
    $fieldMismatches.Count -gt 0 -or
    $missingRouteMappings.Count -gt 0 -or
    $staleRouteMappings.Count -gt 0 -or
    $invalidSchemaRefs.Count -gt 0 -or
    $invalidStatuses.Count -gt 0
) {
    [pscustomobject]@{
        status = 'failed'
        missing_in_doc = $missingInDoc
        stale_in_doc = $staleInDoc
        field_mismatches = $fieldMismatches
        missing_route_mappings = $missingRouteMappings
        stale_route_mappings = $staleRouteMappings
        invalid_schema_refs = $invalidSchemaRefs
        invalid_statuses = $invalidStatuses
    } | ConvertTo-Json -Depth 6
    throw 'API schema doc is out of sync'
}

[pscustomobject]@{
    status = 'passed'
    schema_count = $actualSchemas.Count
    route_mapping_count = $schemaRouteMappings.Count
    doc = $DocPath
    route_doc = $RouteDocPath
    schemas = $actualKeys
} | ConvertTo-Json -Depth 4
