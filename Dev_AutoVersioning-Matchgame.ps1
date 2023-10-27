$utf8WithoutBom = New-Object System.Text.UTF8Encoding $false

# 更新 Dev_fleet.yaml 文件的 imgVersion
$content = [System.IO.File]::ReadAllText('Dev_fleet.yaml', $utf8WithoutBom)
$pattern = 'imgVersion: "(\d+\.\d+\.)(\d+)"'
$match = [regex]::Match($content, $pattern)

if ($match.Success) {
    $oldVersion = $match.Groups[0].Value
    $newVersion = '{0}{1}' -f $match.Groups[1].Value, ([int]$match.Groups[2].Value + 1)
    $content = $content -replace [regex]::Escape($oldVersion), "imgVersion: `"$newVersion`""
    [System.IO.File]::WriteAllText('Dev_fleet.yaml', $content, $utf8WithoutBom)
    Write-Host "Successfully matched and modified the imgVersion in Dev_fleet.yaml to: $newVersion"
} else {
    Write-Host 'Dev_fleet.yaml unmatch'
}

# 更新 Dev_fleet.yaml 文件的 herofishing-matchgame
$content = [System.IO.File]::ReadAllText('Dev_fleet.yaml', $utf8WithoutBom)
$pattern = 'herofishing-matchgame:(\d+\.\d+\.)(\d+)'
$match = [regex]::Match($content, $pattern)

if ($match.Success) {
    $versionMajorMinor = $match.Groups[1].Value
    $versionPatch = [int]$match.Groups[2].Value
    $newVersionPatch = $versionPatch + 1
    $newVersion = $versionMajorMinor + $newVersionPatch
    $content = $content -replace $pattern, ('herofishing-matchgame:' + $newVersion)
    [System.IO.File]::WriteAllText('Dev_fleet.yaml', $content, $utf8WithoutBom)
    Write-Host "Successfully matched and modified the herofishing-matchgame version in Dev_fleet.yaml to: $newVersion"
} else {
    Write-Host 'No matching herofishing-matchgame version found in Dev_fleet.yaml'
}

# 更新 Dev_BuildMatchgame.bat 文件的 herofishing-matchgame
$content = [System.IO.File]::ReadAllText('Dev_BuildMatchgame.bat', $utf8WithoutBom)
$pattern = 'herofishing-matchgame:(\d+\.\d+\.)(\d+)'
$match = [regex]::Match($content, $pattern)

if ($match.Success) {
    $versionMajorMinor = $match.Groups[1].Value
    $versionPatch = [int]$match.Groups[2].Value
    $newVersionPatch = $versionPatch + 1
    $newVersion = $versionMajorMinor + $newVersionPatch
    $content = $content -replace $pattern, ('herofishing-matchgame:' + $newVersion)
    [System.IO.File]::WriteAllText('Dev_BuildMatchgame.bat', $content, $utf8WithoutBom)
    Write-Host "Successfully matched and modified the herofishing-matchgame version in Dev_BuildMatchgame.bat to: $newVersion"
} else {
    Write-Host 'No matching herofishing-matchgame version found in Dev_BuildMatchgame.bat'
}

# 更新 Dev_DeleteAllMatchgameAndKeepByVersion.ps1 文件的要保留版本
$content = [System.IO.File]::ReadAllText('Dev_DeleteAllMatchgameAndKeepByVersion.ps1', $utf8WithoutBom)
$pattern = 'keepVersion = "(\d+\.\d+\.)(\d+)"'
$match = [regex]::Match($content, $pattern)

if ($match.Success) {
    $oldVersion = $match.Groups[0].Value
    $newVersion = '{0}{1}' -f $match.Groups[1].Value, ([int]$match.Groups[2].Value + 1)
    $content = $content -replace [regex]::Escape($oldVersion), "keepVersion = `"$newVersion`""
    [System.IO.File]::WriteAllText('Dev_DeleteAllMatchgameAndKeepByVersion.ps1', $content, $utf8WithoutBom)
    Write-Host "Successfully matched and modified the keepVersion in Dev_DeleteAllMatchgameAndKeepByVersion.ps1 to: $newVersion"
} else {
    Write-Host 'Dev_DeleteAllMatchgameAndKeepByVersion.ps1 unmatch'
}