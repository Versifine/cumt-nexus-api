param(
    [string]$DocPath = 'docs/contracts/http-error-handling.md'
)

$ErrorActionPreference = 'Stop'

$repo = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$docFullPath = Join-Path $repo $DocPath
$apperrPath = Join-Path $repo 'internal/apperr/apperr.go'
$httpErrorPath = Join-Path $repo 'internal/platform/httpserver/error.go'
$responsePath = Join-Path $repo 'internal/platform/httpserver/response.go'

$httpStatusValues = @{
    StatusBadRequest = 400
    StatusUnauthorized = 401
    StatusForbidden = 403
    StatusNotFound = 404
    StatusConflict = 409
    StatusInternalServerError = 500
}

function ConvertTo-Hashtable {
    param([hashtable]$InputObject)

    $result = @{}
    foreach ($key in ($InputObject.Keys | Sort-Object)) {
        $result[$key] = $InputObject[$key]
    }
    return $result
}

if (-not (Test-Path -LiteralPath $docFullPath)) {
    throw "HTTP error contract doc not found: $DocPath"
}

$apperrContent = Get-Content -Raw -Encoding UTF8 -LiteralPath $apperrPath
$codeByConst = @{}
foreach ($match in [regex]::Matches($apperrContent, '(?m)^\s*(Code\w+)\s+Code\s*=\s*"([^"]+)"')) {
    $codeByConst[$match.Groups[1].Value] = $match.Groups[2].Value
}

$httpErrorContent = Get-Content -Raw -Encoding UTF8 -LiteralPath $httpErrorPath
$statusByCode = @{}
foreach ($match in [regex]::Matches($httpErrorContent, '(?s)case\s+apperr\.(Code\w+):\s*return\s+http\.(Status\w+),\s*errorResponse\(err\)')) {
    $constName = $match.Groups[1].Value
    $statusName = $match.Groups[2].Value
    if (-not $codeByConst.ContainsKey($constName)) {
        throw "HTTP error mapper references unknown apperr constant: $constName"
    }
    if (-not $httpStatusValues.ContainsKey($statusName)) {
        throw "HTTP error mapper references unsupported HTTP status: $statusName"
    }
    $statusByCode[$codeByConst[$constName]] = $httpStatusValues[$statusName]
}

if ($codeByConst.ContainsKey('CodeInternal')) {
    $hasInternalFallback = [regex]::IsMatch($httpErrorContent, 'http\.StatusInternalServerError[\s\S]*Code:\s*string\(apperr\.CodeInternal\)[\s\S]*Message:\s*"internal server error"')
    if ($hasInternalFallback) {
        $statusByCode[$codeByConst['CodeInternal']] = 500
    }
}

$responseContent = Get-Content -Raw -Encoding UTF8 -LiteralPath $responsePath
$responseShapeFields = @()
foreach ($match in [regex]::Matches($responseContent, 'json:"([^"]+)"')) {
    $fieldName = $match.Groups[1].Value.Split(',', 2)[0]
    if ($fieldName -ne '' -and $fieldName -ne '-') {
        $responseShapeFields += $fieldName
    }
}

$docContent = Get-Content -Raw -Encoding UTF8 -LiteralPath $docFullPath
$documentedStatusByCode = @{}
foreach ($line in Get-Content -Encoding UTF8 -LiteralPath $docFullPath) {
    $match = [regex]::Match($line, '^\|\s*`([^`]+)`\s*\|\s*(\d+)\s*\|')
    if ($match.Success) {
        $documentedStatusByCode[$match.Groups[1].Value] = [int]$match.Groups[2].Value
    }
}

$codeValues = @($codeByConst.Values | Sort-Object)
$documentedCodes = @($documentedStatusByCode.Keys | Sort-Object)
$actualCodes = @($statusByCode.Keys | Sort-Object)

$missingMapperCodes = @($codeValues | Where-Object { -not $statusByCode.ContainsKey($_) })
$extraMapperCodes = @($actualCodes | Where-Object { $_ -notin $codeValues })
$missingDocCodes = @($codeValues | Where-Object { -not $documentedStatusByCode.ContainsKey($_) })
$staleDocCodes = @($documentedCodes | Where-Object { $_ -notin $codeValues })
$statusMismatches = @()

foreach ($code in $codeValues) {
    if ($statusByCode.ContainsKey($code) -and $documentedStatusByCode.ContainsKey($code) -and $statusByCode[$code] -ne $documentedStatusByCode[$code]) {
        $statusMismatches += [pscustomobject]@{
            code = $code
            actual = $statusByCode[$code]
            documented = $documentedStatusByCode[$code]
        }
    }
}

$requiredShapeFields = @('error', 'code', 'message')
$missingResponseShape = @()
foreach ($field in $requiredShapeFields) {
    $fieldInCode = $field -in $responseShapeFields
    $quotedField = '"' + $field + '"'
    $backtickField = '`' + $field + '`'
    $fieldInDoc = $docContent.Contains($quotedField) -or $docContent.Contains($backtickField)
    if (-not $fieldInCode -or -not $fieldInDoc) {
        $missingResponseShape += $field
    }
}

if (
    $missingMapperCodes.Count -gt 0 -or
    $extraMapperCodes.Count -gt 0 -or
    $missingDocCodes.Count -gt 0 -or
    $staleDocCodes.Count -gt 0 -or
    $statusMismatches.Count -gt 0 -or
    $missingResponseShape.Count -gt 0
) {
    [pscustomobject]@{
        status = 'failed'
        missing_mapper_codes = $missingMapperCodes
        extra_mapper_codes = $extraMapperCodes
        missing_doc_codes = $missingDocCodes
        stale_doc_codes = $staleDocCodes
        status_mismatches = $statusMismatches
        missing_response_shape = $missingResponseShape
    } | ConvertTo-Json -Depth 5
    throw 'HTTP error contract doc is out of sync'
}

[pscustomobject]@{
    status = 'passed'
    doc = $DocPath
    code_count = $codeValues.Count
    response_shape = $requiredShapeFields
    mappings = ConvertTo-Hashtable -InputObject $statusByCode
} | ConvertTo-Json -Depth 4
