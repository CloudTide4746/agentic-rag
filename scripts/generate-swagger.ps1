$ErrorActionPreference = 'Stop'

Push-Location (Split-Path -Parent $PSScriptRoot)
try {
    $swag = Get-Command swag -ErrorAction SilentlyContinue
    if (-not $swag) {
        throw '未找到 swag。请先运行: go install github.com/swaggo/swag/cmd/swag@v1.16.6'
    }

    & $swag.Source init `
        --generalInfo cmd/main.go `
        --dir . `
        --output docs `
        --outputTypes json,yaml `
        --parseInternal
    if ($LASTEXITCODE -ne 0) {
        throw "swag init 失败，退出码: $LASTEXITCODE"
    }
}
finally {
    Pop-Location
}
