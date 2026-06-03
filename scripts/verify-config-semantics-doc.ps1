param(
    [string]$DocPath = 'docs/internal/engineering/configuration.md'
)

$ErrorActionPreference = 'Stop'

$repo = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$loadPath = Join-Path $repo 'internal/platform/config/load.go'
$validatePath = Join-Path $repo 'internal/platform/config/validate.go'
$docFullPath = Join-Path $repo $DocPath

function Sorted {
    param([System.Collections.IEnumerable]$Values)
    return @($Values | Sort-Object)
}

function Normalize-DocCell {
    param([string]$Value)
    return ($Value -replace '<br\s*/?>', ' ' -replace '\s+', ' ').Trim()
}

function Remove-Backticks {
    param([string]$Value)
    return ($Value -replace '`', '').Trim()
}

function Normalize-RequiredCell {
    param([string]$Value)

    $normalized = Normalize-DocCell $Value
    $yes = [string][char]0x662F
    $no = [string][char]0x5426
    $requiredWord = ([string][char]0x5FC5) + ([string][char]0x9700)

    if ($normalized -eq $yes) {
        return 'yes'
    }
    if ($normalized -eq $no) {
        return 'no'
    }
    if ($normalized -match '^`(?<provider>r2|local)`\s*' + [regex]::Escape($requiredWord) + '$') {
        return "provider:$($Matches['provider'])"
    }
    return (Remove-Backticks $normalized)
}

function Normalize-DefaultCell {
    param([string]$Value)

    $normalized = Normalize-DocCell $Value
    $none = [string][char]0x65E0
    $empty = [string][char]0x7A7A
    $emptyDefaultPrefix = 'local ' + $empty + ([string][char]0x503C) + ([string][char]0x65F6) + ([string][char]0x8865) + ' '

    if ($normalized -eq $none) {
        return 'none'
    }
    if ($normalized -eq $empty) {
        return 'empty'
    }
    if ($normalized.StartsWith($emptyDefaultPrefix)) {
        $url = ($normalized.Substring($emptyDefaultPrefix.Length) -replace '`', '').Trim()
        return "local-empty-default:$url"
    }
    return (Remove-Backticks $normalized)
}

function Convert-GoDefaultExpression {
    param([string]$Expression)

    $value = $Expression.Trim()
    if ($value -match '^"(?<text>.*)"$') {
        $text = $Matches['text']
        if ($text -eq '') {
            return 'none'
        }
        return $text
    }
    if ($value -eq 'nil') {
        return 'empty'
    }
    if ($value -eq 'true' -or $value -eq 'false') {
        return $value
    }
    if ($value -match '^(?<n>\d+)\s*\*\s*time\.(?<unit>Second|Minute|Hour)$') {
        $n = [int]$Matches['n']
        switch ($Matches['unit']) {
            'Second' { return "$($n)s" }
            'Minute' { return "$($n)m" }
            'Hour' { return "$($n)h" }
        }
    }
    if ($value -match '^(?<a>\d+)\s*\*\s*(?<b>\d+)\s*\*\s*(?<c>\d+)$') {
        return [string]([int]$Matches['a'] * [int]$Matches['b'] * [int]$Matches['c'])
    }
    if ($value -match '^\d+$') {
        return $value
    }

    throw "unsupported default expression: $Expression"
}

if (-not (Test-Path -LiteralPath $loadPath)) {
    throw 'config loader not found'
}
if (-not (Test-Path -LiteralPath $validatePath)) {
    throw 'config validator not found'
}
if (-not (Test-Path -LiteralPath $docFullPath)) {
    throw "configuration doc not found: $DocPath"
}

$loadContent = Get-Content -Raw -Encoding UTF8 -LiteralPath $loadPath
$validateContent = Get-Content -Raw -Encoding UTF8 -LiteralPath $validatePath

$expected = @{}

foreach ($match in [regex]::Matches($loadContent, 'requiredString\("(?<key>[A-Z0-9_]+)"')) {
    $expected[$match.Groups['key'].Value] = [pscustomobject]@{
        key = $match.Groups['key'].Value
        required = 'yes'
        default = 'none'
        enum_values = @()
    }
}

$defaultPatterns = @(
    '(?:stringDefault|stringListDefault|intDefault|boolDefault|durationDefault)\("(?<key>[A-Z0-9_]+)",\s*(?<default>[^,\r\n\)]+)'
)

foreach ($pattern in $defaultPatterns) {
    foreach ($match in [regex]::Matches($loadContent, $pattern)) {
        $key = $match.Groups['key'].Value
        $expected[$key] = [pscustomobject]@{
            key = $key
            required = 'no'
            default = (Convert-GoDefaultExpression -Expression $match.Groups['default'].Value)
            enum_values = @()
        }
    }
}

if ([regex]::IsMatch($loadContent, 'cfg\.Storage\.Provider\s*==\s*"local"\s*&&\s*cfg\.Storage\.PublicBaseURL\s*==\s*""[\s\S]*?cfg\.Storage\.PublicBaseURL\s*=\s*"http://localhost:8080/uploads"')) {
    $expected['OBJECT_STORAGE_PUBLIC_BASE_URL'].default = 'local-empty-default:http://localhost:8080/uploads'
}

foreach ($match in [regex]::Matches($validateContent, '(?<key>[A-Z0-9_]+) must be one of (?<values>[a-z0-9_\-/]+)')) {
    $key = $match.Groups['key'].Value
    if ($expected.ContainsKey($key)) {
        $expected[$key].enum_values = @($match.Groups['values'].Value.Split('/') | Where-Object { $_ -ne '' })
    }
}

foreach ($match in [regex]::Matches($validateContent, '(?<key>[A-Z0-9_]+) is required for r2 storage')) {
    $key = $match.Groups['key'].Value
    if ($expected.ContainsKey($key) -and $expected[$key].default -eq 'none') {
        $expected[$key].required = 'provider:r2'
    }
}

foreach ($match in [regex]::Matches($validateContent, '(?<key>[A-Z0-9_]+) cannot be empty for local storage')) {
    $key = $match.Groups['key'].Value
    if ($expected.ContainsKey($key) -and $expected[$key].default -eq 'none') {
        $expected[$key].required = 'provider:local'
    }
}

if ($expected.ContainsKey('OBJECT_STORAGE_PUBLIC_BASE_URL')) {
    $expected['OBJECT_STORAGE_PUBLIC_BASE_URL'].required = 'provider:r2'
}

$documented = @{}
foreach ($line in Get-Content -Encoding UTF8 -LiteralPath $docFullPath) {
    $match = [regex]::Match($line, '^\|\s*`(?<key>[A-Z0-9_]+)`\s*\|\s*(?<required>[^|]+?)\s*\|\s*(?<default>[^|]+?)\s*(?:\|\s*(?<description>[^|]*?))?\s*\|$')
    if (-not $match.Success) {
        continue
    }

    $key = $match.Groups['key'].Value
    $documented[$key] = [pscustomobject]@{
        key = $key
        required = Normalize-RequiredCell $match.Groups['required'].Value
        default = Normalize-DefaultCell $match.Groups['default'].Value
        description = Normalize-DocCell $match.Groups['description'].Value
    }
}

$missingInDoc = [System.Collections.Generic.List[string]]::new()
$staleInDoc = [System.Collections.Generic.List[string]]::new()
$requiredMismatches = [System.Collections.Generic.List[object]]::new()
$defaultMismatches = [System.Collections.Generic.List[object]]::new()
$enumMismatches = [System.Collections.Generic.List[object]]::new()

foreach ($key in $expected.Keys) {
    if (-not $documented.ContainsKey($key)) {
        $missingInDoc.Add($key) | Out-Null
        continue
    }

    $actual = $documented[$key]
    $want = $expected[$key]
    if ($actual.required -ne $want.required) {
        $requiredMismatches.Add([pscustomobject]@{
            key = $key
            expected = $want.required
            actual = $actual.required
        }) | Out-Null
    }
    if ($actual.default -ne $want.default) {
        $defaultMismatches.Add([pscustomobject]@{
            key = $key
            expected = $want.default
            actual = $actual.default
        }) | Out-Null
    }
    foreach ($enumValue in $want.enum_values) {
        if ($actual.description -notmatch "(^|[^a-z0-9_-])$([regex]::Escape($enumValue))([^a-z0-9_-]|$)") {
            $enumMismatches.Add([pscustomobject]@{
                key = $key
                missing_enum_value = $enumValue
                description = $actual.description
            }) | Out-Null
        }
    }
}

foreach ($key in $documented.Keys) {
    if (-not $expected.ContainsKey($key)) {
        $staleInDoc.Add($key) | Out-Null
    }
}

if (
    $missingInDoc.Count -gt 0 -or
    $staleInDoc.Count -gt 0 -or
    $requiredMismatches.Count -gt 0 -or
    $defaultMismatches.Count -gt 0 -or
    $enumMismatches.Count -gt 0
) {
    [pscustomobject]@{
        status = 'failed'
        missing_in_doc = Sorted $missingInDoc
        stale_in_doc = Sorted $staleInDoc
        required_mismatches = @($requiredMismatches)
        default_mismatches = @($defaultMismatches)
        enum_mismatches = @($enumMismatches)
    } | ConvertTo-Json -Depth 5
    throw 'configuration semantic contract is out of sync'
}

[pscustomobject]@{
    status = 'passed'
    checked_key_count = $expected.Count
    enum_key_count = @($expected.Values | Where-Object { $_.enum_values.Count -gt 0 }).Count
    doc = $DocPath
    keys = Sorted $expected.Keys
} | ConvertTo-Json -Depth 5
