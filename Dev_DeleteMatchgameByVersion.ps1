# 指定要刪除pod的label版本與pod所在命名空間
$target_version = "0.1.22"
$namespace = "herofishing-gameserver"

$removedPodsCount = 0 # 移除pod數量

# 取得所有pod名稱
$pods = & kubectl get pods --namespace=$namespace --selector=imgVersion=$target_version -o jsonpath="{.items[*].metadata.name}"
if ($LASTEXITCODE -ne 0) {
    Write-Host "An error occurred while fetching pods."
    exit 1
}
Write-Host "Got Pods: $pods"

# 如果沒有取到pod就退出
if (-not $pods) {
    Write-Host "No Pods found matching the criteria."
    exit 0
}

$pods = $pods -split " "  #建立pod陣列

# 遍歷並移除pods
foreach ($pod in $pods) {
    Write-Host "Ready to remove Pod: $pod"
    
    # 刪除該pod
    # & kubectl delete pod $pod --namespace=$namespace | Out-Null
    # if ($LASTEXITCODE -ne 0) {
    #     Write-Host "An error occurred while deleting the pod."
    #     exit 1
    # }


    # 非同步刪除該 pod
    Start-Process -NoNewWindow -FilePath "kubectl" -ArgumentList "delete pod $pod --namespace=$namespace"

    $removedPodsCount++
}
Write-Host "$removedPodsCount Pods have been Removed"