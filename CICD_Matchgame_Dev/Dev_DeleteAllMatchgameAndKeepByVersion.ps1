# 指定要刪除的pod類型與所在命名空間, 並指定要保留的版本(非指定版本都會刪除)
$keepVersion = "0.2.322"  
$type = "herofishing-matchgame"
$namespace = "herofishing-gameserver"

$removedPodsCount = 0 # 已移除的pod數量

# 獲取所有符合 type 的 pod 名稱和版本
$pods = & kubectl get pods --namespace=$namespace --selector=type=$type -o jsonpath="{range .items[*]}{@.metadata.name}:{@.metadata.labels.imgVersion} {end}"
if ($LASTEXITCODE -ne 0) {
    Write-Host "Error: get pods failed"
    exit 1
}

# 如果沒有取到pod就退出
if (-not $pods) {
    Write-Host "No pod found"
    exit 0
}

$pods = $pods.Trim() -split " " #將結果分割成一個陣列
Write-Host "Parsed Pods: $pods"

# 遍歷並判斷是否移除pods
foreach ($pod_info in $pods) {
    $pod_name, $pod_version = $pod_info -split ":"
    if ($pod_version -ne $keepVersion) {
        Write-Host "Ready to remove Pod: $pod_name"

        # 非同步刪除該 pod
        Start-Process -NoNewWindow -FilePath "kubectl" -ArgumentList "delete pod $pod_name --namespace=$namespace" -RedirectStandardOutput "NUL"

        $removedPodsCount++
    }
}