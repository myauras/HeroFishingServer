# 指定要刪除pod的image版本與pod所在命名空間
$target_version = "0.1.6"
$namespace = "herofishing-gameserver"

# 獲取所有 pod 名稱
$pods = & kubectl get pods --namespace=$namespace --selector=your-label-selector -o jsonpath="{.items[*].metadata.name}"
Write-Host "Got Pods: $pods"
$pods = $pods -split " "  # 分割多個 pod 名稱為陣列

# 遍歷每個 pod
foreach ($pod in $pods) {
    # 獲取 pod 的 image 版本
    $version = & kubectl get pod $pod --namespace=$namespace -o=jsonpath='{.spec.containers[*].image}'
    Write-Host "Pod: $pod, Version: $version"
    
    # 檢查 image 版本是否匹配目標版本
    if ($version -like "*$target_version*") {
        Write-Host "Version match! Ready to remove Pod: $pod"
        # 刪除該 pod
        & kubectl delete pod $pod --namespace=$namespace
    }
}
