$utf8WithoutBom = New-Object System.Text.UTF8Encoding $false

# 更新 Dev_Crontasker.yaml 文件的版本
$content = [System.IO.File]::ReadAllText('Dev_Crontasker.yaml', $utf8WithoutBom)
$pattern = 'herofishing-crontasker:(\d+\.\d+\.)(\d+)'
$match = [regex]::Match($content, $pattern)

if ($match.Success) {
    $versionMajorMinor = $match.Groups[1].Value
    $versionPatch = [int]$match.Groups[2].Value
    $newVersionPatch = $versionPatch + 1
    $newVersion = $versionMajorMinor + $newVersionPatch
    $content = $content -replace $pattern, ('herofishing-crontasker:' + $newVersion)
    [System.IO.File]::WriteAllText('Dev_Crontasker.yaml', $content, $utf8WithoutBom)
    Write-Host "Successfully matched and modified the version to: $newVersion"
} else {
    Write-Host 'No matching version found for herofishing-crontasker in Dev_Crontasker.yaml'
}

# 更新 Dev_BuildCrontasker.bat 文件的版本
$content = [System.IO.File]::ReadAllText('Dev_BuildCrontasker.bat', $utf8WithoutBom)
$pattern = 'herofishing-crontasker:(\d+\.\d+\.)(\d+)'
$match = [regex]::Match($content, $pattern)

if ($match.Success) {
    $versionMajorMinor = $match.Groups[1].Value
    $versionPatch = [int]$match.Groups[2].Value
    $newVersionPatch = $versionPatch + 1
    $newVersion = $versionMajorMinor + $newVersionPatch
    $content = $content -replace $pattern, ('herofishing-crontasker:' + $newVersion)
    [System.IO.File]::WriteAllText('Dev_BuildCrontasker.bat', $content, $utf8WithoutBom)
    Write-Host "Successfully matched and modified the version to: $newVersion"
} else {
    Write-Host 'No matching version found for herofishing-crontasker in Dev_BuildCrontasker.bat'
}
