@echo off
powershell -NoProfile -ExecutionPolicy Bypass -command "$utf8WithoutBom = New-Object System.Text.UTF8Encoding $false; $content = [System.IO.File]::ReadAllText('Dev_Matchmaker.yaml', $utf8WithoutBom); $pattern = 'herofishing-matchmaker:(\d+\.\d+\.)(\d+)'; $match = [regex]::Match($content, $pattern); if ($match.Success) { $versionMajorMinor = $match.Groups[1].Value; $versionPatch = [int]$match.Groups[2].Value; $newVersionPatch = $versionPatch + 1; $newVersion = $versionMajorMinor + $newVersionPatch; $content = $content -replace $pattern, ('herofishing-matchmaker:' + $newVersion); [System.IO.File]::WriteAllText('Dev_Matchmaker.yaml', $content, $utf8WithoutBom); }"
powershell -NoProfile -ExecutionPolicy Bypass -command "$utf8WithoutBom = New-Object System.Text.UTF8Encoding $false; $content = [System.IO.File]::ReadAllText('Dev_BuildMatchmaker.bat', $utf8WithoutBom); $pattern = 'herofishing-matchmaker:(\d+\.\d+\.)(\d+)'; $match = [regex]::Match($content, $pattern); if ($match.Success) { $versionMajorMinor = $match.Groups[1].Value; $versionPatch = [int]$match.Groups[2].Value; $newVersionPatch = $versionPatch + 1; $newVersion = $versionMajorMinor + $newVersionPatch; $content = $content -replace $pattern, ('herofishing-matchmaker:' + $newVersion); [System.IO.File]::WriteAllText('Dev_BuildMatchmaker.bat', $content, $utf8WithoutBom); }"
@if ERRORLEVEL 1 exit /b 1
@echo on