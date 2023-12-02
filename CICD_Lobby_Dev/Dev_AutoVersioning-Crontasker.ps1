$utf8WithoutBom = New-Object System.Text.UTF8Encoding $false

# 更新 Dev_Lobby.yaml 文件的版本
$content = [System.IO.File]::ReadAllText('Dev_Lobby.yaml', $utf8WithoutBom)
$pattern = 'herofishing-lobby:(\d+\.\d+\.)(\d+)'
$match = [regex]::Match($content, $pattern)

if ($match.Success) {
    $versionMajorMinor = $match.Groups[1].Value
    $versionPatch = [int]$match.Groups[2].Value
    $newVersionPatch = $versionPatch + 1
    $newVersion = $versionMajorMinor + $newVersionPatch
    $content = $content -replace $pattern, ('herofishing-lobby:' + $newVersion)
    [System.IO.File]::WriteAllText('Dev_Lobby.yaml', $content, $utf8WithoutBom)
    Write-Host "Successfully matched and modified the version to: $newVersion"
} else {
    Write-Host 'No matching version found for herofishing-lobby in Dev_Lobby.yaml'
}

# 更新 Dev_Lobby.yaml 文件的 imgVersion
$content = [System.IO.File]::ReadAllText('Dev_Lobby.yaml', $utf8WithoutBom)
$pattern = 'imgVersion: "(\d+\.\d+\.)(\d+)"'
$match = [regex]::Match($content, $pattern)

if ($match.Success) {
    $oldVersion = $match.Groups[0].Value
    $newVersion = '{0}{1}' -f $match.Groups[1].Value, ([int]$match.Groups[2].Value + 1)
    $content = $content -replace [regex]::Escape($oldVersion), "imgVersion: `"$newVersion`""
    [System.IO.File]::WriteAllText('Dev_Lobby.yaml', $content, $utf8WithoutBom)
    Write-Host "Successfully matched and modified the imgVersion in Dev_Lobby.yaml to: $newVersion"
} else {
    Write-Host 'Dev_Lobby.yaml unmatch'
}

# 更新 Dev_BuildLobby.bat 文件的版本
$content = [System.IO.File]::ReadAllText('Dev_BuildLobby.bat', $utf8WithoutBom)
$pattern = 'herofishing-lobby:(\d+\.\d+\.)(\d+)'
$match = [regex]::Match($content, $pattern)

if ($match.Success) {
    $versionMajorMinor = $match.Groups[1].Value
    $versionPatch = [int]$match.Groups[2].Value
    $newVersionPatch = $versionPatch + 1
    $newVersion = $versionMajorMinor + $newVersionPatch
    $content = $content -replace $pattern, ('herofishing-lobby:' + $newVersion)
    [System.IO.File]::WriteAllText('Dev_BuildLobby.bat', $content, $utf8WithoutBom)
    Write-Host "Successfully matched and modified the version to: $newVersion"
} else {
    Write-Host 'No matching version found for herofishing-lobby in Dev_BuildLobby.bat'
}


# 更新 Dev_DeleteAllLobbyAndKeepByVersion.ps1 文件的要保留版本
$content = [System.IO.File]::ReadAllText('Dev_DeleteAllLobbyAndKeepByVersion.ps1', $utf8WithoutBom)
$pattern = 'keepVersion = "(\d+\.\d+\.)(\d+)"'
$match = [regex]::Match($content, $pattern)

if ($match.Success) {
    $oldVersion = $match.Groups[0].Value
    $newVersion = '{0}{1}' -f $match.Groups[1].Value, ([int]$match.Groups[2].Value + 1)
    $content = $content -replace [regex]::Escape($oldVersion), "keepVersion = `"$newVersion`""
    [System.IO.File]::WriteAllText('Dev_DeleteAllLobbyAndKeepByVersion.ps1', $content, $utf8WithoutBom)
    Write-Host "Successfully matched and modified the keepVersion in Dev_DeleteAllLobbyAndKeepByVersion.ps1 to: $newVersion"
} else {
    Write-Host 'Dev_DeleteAllLobbyAndKeepByVersion.ps1 unmatch'
}