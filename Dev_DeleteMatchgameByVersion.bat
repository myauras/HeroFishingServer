@REM 要刪除的image版本
$target_version = "0.1.6"
$pods = kubectl get pods --all-namespaces --selector=your-label-selector -o jsonpath="{.items[*].metadata.name}"

foreach ($pod in $pods) {
    $version = kubectl get pod $pod -o=jsonpath='{.spec.containers[*].image}'
    if ($version -like "*$target_version*") {
        kubectl delete pod $pod
    }
}
