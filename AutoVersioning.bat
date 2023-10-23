@echo off
setlocal

:: 執行 PowerShell 腳本
powershell -NoProfile -ExecutionPolicy Bypass -command ^
" 
    $content = Get-Content -Path 'Dev_Matchmaker.yaml' -Raw
    $pattern = 'herofishing-matchmaker:(\d+\.\d+\.)(\d+)'

    $match = [regex]::Match($content, $pattern)

    if ($match.Success) {
        $versionMajorMinor = $match.Groups[1].Value
        $versionPatch = [int]$match.Groups[2].Value
        $newVersionPatch = $versionPatch + 1
        $newVersion = "" + $versionMajorMinor + $newVersionPatch

        $content = $content -replace $pattern, ('herofishing-matchmaker:' + $newVersion)
        Set-Content -Path 'Dev_Matchmaker.yaml' -Value $content
    }
"

endlocal
