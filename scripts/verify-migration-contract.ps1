param(
    [string]$MigrationDir = 'migrations',
    [string]$DocPath = 'docs/contracts/migrations.md'
)

$ErrorActionPreference = 'Stop'

$repo = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$migrationFullPath = Join-Path $repo $MigrationDir
$docFullPath = Join-Path $repo $DocPath

function Sorted {
    param([System.Collections.IEnumerable]$Values)
    return @($Values | Sort-Object)
}

function New-MigrationKey {
    param(
        [string]$Version,
        [string]$Name
    )
    return "$Version`_$Name"
}

if (-not (Test-Path -LiteralPath $migrationFullPath)) {
    throw "migration directory not found: $MigrationDir"
}
if (-not (Test-Path -LiteralPath $docFullPath)) {
    throw "migration contract doc not found: $DocPath"
}

$pattern = '^(?<version>\d{6})_(?<name>[a-z0-9_]+)\.(?<direction>up|down)\.sql$'
$entries = [System.Collections.Generic.List[object]]::new()
$unexpectedFiles = [System.Collections.Generic.List[string]]::new()

foreach ($file in Get-ChildItem -LiteralPath $migrationFullPath -File) {
    $match = [regex]::Match($file.Name, $pattern)
    if (-not $match.Success) {
        $unexpectedFiles.Add($file.Name) | Out-Null
        continue
    }

    $entries.Add([pscustomobject]@{
        version = $match.Groups['version'].Value
        version_number = [int]$match.Groups['version'].Value
        name = $match.Groups['name'].Value
        direction = $match.Groups['direction'].Value
        file = $file.Name
    }) | Out-Null
}

if ($entries.Count -eq 0) {
    throw 'no migration files found'
}

$pairErrors = [System.Collections.Generic.List[object]]::new()
$contiguityErrors = [System.Collections.Generic.List[object]]::new()
$actualInventory = [System.Collections.Generic.List[object]]::new()

$groups = @($entries | Group-Object -Property version | Sort-Object Name)
$versions = @($groups | ForEach-Object { [int]$_.Name })
for ($i = 0; $i -lt $versions.Count; $i++) {
    $expected = $i + 1
    if ($versions[$i] -ne $expected) {
        $contiguityErrors.Add([pscustomobject]@{
            expected = ('{0:D6}' -f $expected)
            actual = ('{0:D6}' -f $versions[$i])
        }) | Out-Null
    }
}

foreach ($group in $groups) {
    $items = @($group.Group)
    $upFiles = @($items | Where-Object { $_.direction -eq 'up' })
    $downFiles = @($items | Where-Object { $_.direction -eq 'down' })

    if ($upFiles.Count -ne 1 -or $downFiles.Count -ne 1) {
        $pairErrors.Add([pscustomobject]@{
            version = $group.Name
            up_files = @($upFiles | ForEach-Object { $_.file })
            down_files = @($downFiles | ForEach-Object { $_.file })
        }) | Out-Null
        continue
    }

    if ($upFiles[0].name -ne $downFiles[0].name) {
        $pairErrors.Add([pscustomobject]@{
            version = $group.Name
            up_name = $upFiles[0].name
            down_name = $downFiles[0].name
        }) | Out-Null
        continue
    }

    $actualInventory.Add([pscustomobject]@{
        version = $upFiles[0].version
        name = $upFiles[0].name
        up = $upFiles[0].file
        down = $downFiles[0].file
    }) | Out-Null
}

$documentedInventory = [System.Collections.Generic.List[object]]::new()
$docVersionCounts = @{}
foreach ($line in Get-Content -Encoding UTF8 -LiteralPath $docFullPath) {
    $match = [regex]::Match($line, '^\|\s*(?<version>\d{6})\s*\|\s*(?<name>[a-z0-9_]+)\s*\|')
    if (-not $match.Success) {
        continue
    }

    $version = $match.Groups['version'].Value
    $name = $match.Groups['name'].Value
    $documentedInventory.Add([pscustomobject]@{
        version = $version
        name = $name
        key = (New-MigrationKey -Version $version -Name $name)
    }) | Out-Null

    if (-not $docVersionCounts.ContainsKey($version)) {
        $docVersionCounts[$version] = 0
    }
    $docVersionCounts[$version]++
}

$actualByVersion = @{}
foreach ($item in $actualInventory) {
    $actualByVersion[$item.version] = $item.name
}

$docByVersion = @{}
foreach ($item in $documentedInventory) {
    if (-not $docByVersion.ContainsKey($item.version)) {
        $docByVersion[$item.version] = $item.name
    }
}

$duplicateDocumentedVersions = @($docVersionCounts.GetEnumerator() |
    Where-Object { $_.Value -gt 1 } |
    ForEach-Object { $_.Key } |
    Sort-Object)

$missingInDoc = [System.Collections.Generic.List[string]]::new()
$staleInDoc = [System.Collections.Generic.List[string]]::new()
$docNameMismatches = [System.Collections.Generic.List[object]]::new()

foreach ($item in $actualInventory) {
    if (-not $docByVersion.ContainsKey($item.version)) {
        $missingInDoc.Add((New-MigrationKey -Version $item.version -Name $item.name)) | Out-Null
        continue
    }

    if ($docByVersion[$item.version] -ne $item.name) {
        $docNameMismatches.Add([pscustomobject]@{
            version = $item.version
            actual_name = $item.name
            documented_name = $docByVersion[$item.version]
        }) | Out-Null
    }
}

foreach ($item in $documentedInventory) {
    if (-not $actualByVersion.ContainsKey($item.version)) {
        $staleInDoc.Add($item.key) | Out-Null
    }
}

if (
    $unexpectedFiles.Count -gt 0 -or
    $pairErrors.Count -gt 0 -or
    $contiguityErrors.Count -gt 0 -or
    $duplicateDocumentedVersions.Count -gt 0 -or
    $missingInDoc.Count -gt 0 -or
    $staleInDoc.Count -gt 0 -or
    $docNameMismatches.Count -gt 0
) {
    [pscustomobject]@{
        status = 'failed'
        unexpected_files = Sorted $unexpectedFiles
        pair_errors = @($pairErrors)
        contiguity_errors = @($contiguityErrors)
        duplicate_documented_versions = $duplicateDocumentedVersions
        missing_in_doc = Sorted $missingInDoc
        stale_in_doc = Sorted $staleInDoc
        doc_name_mismatches = @($docNameMismatches)
    } | ConvertTo-Json -Depth 5
    throw 'migration contract is out of sync'
}

[pscustomobject]@{
    status = 'passed'
    migration_count = $actualInventory.Count
    doc = $DocPath
    migrations = @($actualInventory | Sort-Object version)
} | ConvertTo-Json -Depth 5
