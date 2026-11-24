# 兼容旧命令；配置加载与校验统一由 Go 完成。
$ErrorActionPreference = 'Stop'
$apiRoot = Split-Path -Parent $PSScriptRoot
Push-Location $apiRoot
try {
    go run ./cmd/api -config config.yaml
    $runExitCode = $LASTEXITCODE
}
finally {
    Pop-Location
}
exit $runExitCode
