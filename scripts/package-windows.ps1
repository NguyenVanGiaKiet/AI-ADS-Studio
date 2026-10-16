$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent $PSScriptRoot
$frontendDir = Join-Path $repoRoot "frontend"
$backendDir = Join-Path $repoRoot "backend"
$modelsDir = Join-Path $backendDir "models"
$releaseDir = Join-Path $backendDir "release"
$stageDir = Join-Path $env:TEMP ("ai-ads-studio-" + [guid]::NewGuid().ToString("N"))
$webDir = Join-Path $stageDir "web"
$runtimeDir = Join-Path $stageDir "runtime"
$ffmpegDir = Join-Path $stageDir "ffmpeg"
$innoCompiler = Join-Path $env:LOCALAPPDATA "Programs\Inno Setup 6\ISCC.exe"
$innoTemplate = Join-Path $PSScriptRoot "windows\installer.iss"
$pythonRuntimeVersion = "3.13.4"
$pythonRuntimeArchive = "python-$pythonRuntimeVersion-embed-amd64.zip"
$pythonRuntimeUrl = "https://www.python.org/ftp/python/$pythonRuntimeVersion/$pythonRuntimeArchive"
$pythonExecutable = (Get-Command python -ErrorAction Stop).Source
$ffmpegExecutable = (Get-Command ffmpeg -ErrorAction Stop).Source
$ffprobeExecutable = (Get-Command ffprobe -ErrorAction Stop).Source

foreach ($command in @("go", "npm", "python")) {
    if (-not (Get-Command $command -ErrorAction SilentlyContinue)) {
        throw "Required build tool '$command' was not found in PATH."
    }
}
if (-not (Test-Path $innoCompiler)) {
    throw "Inno Setup compiler was not found at '$innoCompiler'. Install Inno Setup 6 first."
}
if (-not (Test-Path $innoTemplate)) {
    throw "Inno Setup script template is missing: $innoTemplate"
}

$requiredModels = @(
    "vi_VN-vais1000-medium.onnx",
    "vi_VN-vais1000-medium.onnx.json",
    "vi_VN-25hours_single-low.onnx",
    "vi_VN-25hours_single-low.onnx.json",
    "vi_VN-csa-voice-piper-v3-medium.onnx",
    "vi_VN-csa-voice-piper-v3-medium.onnx.json"
)
foreach ($model in $requiredModels) {
    if (-not (Test-Path (Join-Path $modelsDir $model))) {
        throw "Required voice model file is missing: $model"
    }
}

New-Item -ItemType Directory -Path $stageDir | Out-Null
try {
    Push-Location $frontendDir
    try {
        npm run build
        if ($LASTEXITCODE -ne 0) {
            throw "Frontend build failed."
        }
    }
    finally {
        Pop-Location
    }

    New-Item -ItemType Directory -Path $webDir | Out-Null
    Copy-Item (Join-Path $frontendDir "dist\*") $webDir -Recurse -Force

    Push-Location $backendDir
    try {
        go build -trimpath -ldflags="-s -w" -o (Join-Path $stageDir "AI-ADS-Studio.exe") .
        if ($LASTEXITCODE -ne 0) {
            throw "Backend build failed."
        }
    }
    finally {
        Pop-Location
    }

    $stageModelsDir = Join-Path $stageDir "models"
    New-Item -ItemType Directory -Path $stageModelsDir | Out-Null
    foreach ($model in $requiredModels) {
        Copy-Item (Join-Path $modelsDir $model) $stageModelsDir
    }
    Copy-Item (Join-Path $frontendDir "public\ai-ads-studio.ico") $stageDir

    $pythonArchivePath = Join-Path $stageDir $pythonRuntimeArchive
    Invoke-WebRequest -Uri $pythonRuntimeUrl -OutFile $pythonArchivePath
    Expand-Archive -LiteralPath $pythonArchivePath -DestinationPath $runtimeDir
    Remove-Item -LiteralPath $pythonArchivePath -Force

    $pythonPth = Join-Path $runtimeDir "python313._pth"
    if (-not (Test-Path $pythonPth)) {
        throw "The embedded Python runtime did not contain python313._pth."
    }
    @"
python313.zip
.
Lib\site-packages
import site
"@ | Set-Content -Path $pythonPth -Encoding ascii

    $sitePackages = Join-Path $runtimeDir "Lib\site-packages"
    New-Item -ItemType Directory -Path $sitePackages -Force | Out-Null
    & $pythonExecutable -m pip install --disable-pip-version-check --no-warn-script-location --target $sitePackages -r (Join-Path $backendDir "requirements.txt")
    if ($LASTEXITCODE -ne 0) {
        throw "Installing the bundled Python runtime dependencies failed."
    }

    New-Item -ItemType Directory -Path $ffmpegDir | Out-Null
    Copy-Item $ffmpegExecutable $ffmpegDir -Force
    Copy-Item $ffprobeExecutable $ffmpegDir -Force
    $ffmpegRoot = Split-Path (Split-Path $ffmpegExecutable -Parent) -Parent
    foreach ($licenseFile in @("LICENSE", "README.txt")) {
        $licensePath = Join-Path $ffmpegRoot $licenseFile
        if (Test-Path $licensePath) {
            Copy-Item $licensePath (Join-Path $ffmpegDir ("FFmpeg-" + $licenseFile)) -Force
        }
    }

    $backendEnv = Join-Path $backendDir ".env"
    if (-not (Test-Path $backendEnv)) {
        throw "The current backend/.env is required to package the configured Groq key."
    }
    $envLines = Get-Content $backendEnv
    $groqKeyLine = $envLines | Where-Object { $_ -match '^\s*GROQ_API_KEY\s*=\s*(.+?)\s*$' } | Select-Object -First 1
    if (-not $groqKeyLine) {
        throw "No GROQ_API_KEY is configured in backend/.env."
    }
    $groqKey = [regex]::Match($groqKeyLine, '^\s*GROQ_API_KEY\s*=\s*(.+?)\s*$').Groups[1].Value.Trim().Trim([char[]]@('"', "'"))
    if (-not $groqKey -or $groqKey -match '^(your_|replace-|changeme)') {
        throw "GROQ_API_KEY in backend/.env is empty or still a placeholder."
    }
    if ($groqKey.Contains("`r") -or $groqKey.Contains("`n")) {
        throw "GROQ_API_KEY must be a single-line value."
    }
    Set-Content -Path (Join-Path $stageDir "groq-key.txt") -Value $groqKey -NoNewline -Encoding ascii
    $groqModelLine = $envLines | Where-Object { $_ -match '^\s*GROQ_MODEL\s*=\s*(.+?)\s*$' } | Select-Object -First 1
    $groqModel = if ($groqModelLine) {
        [regex]::Match($groqModelLine, '^\s*GROQ_MODEL\s*=\s*(.+?)\s*$').Groups[1].Value.Trim().Trim([char[]]@('"', "'"))
    } else {
        "qwen/qwen3.8-27b"
    }
    if ($groqModel.Contains("`r") -or $groqModel.Contains("`n")) {
        throw "GROQ_MODEL must be a single-line value."
    }

    Copy-Item (Join-Path $PSScriptRoot "windows\Start.cmd") $stageDir
    Copy-Item (Join-Path $PSScriptRoot "windows\README.txt") $stageDir
    $licenseDir = Join-Path $stageDir "licenses"
    New-Item -ItemType Directory -Path $licenseDir | Out-Null
    $pythonLicense = Join-Path $runtimeDir "LICENSE.txt"
    if (Test-Path $pythonLicense) {
        Copy-Item $pythonLicense (Join-Path $licenseDir "Python-LICENSE.txt")
    }

    New-Item -ItemType Directory -Path $releaseDir -Force | Out-Null
    $issPath = Join-Path $stageDir "AI-ADS-Studio.iss"
    $escapedGroqModel = $groqModel.Replace('"', '""')
    $scriptContent = Get-Content $innoTemplate -Raw
    $scriptContent = $scriptContent.Replace("{{STAGE_DIR}}", $stageDir)
    $scriptContent = $scriptContent.Replace("{{OUTPUT_DIR}}", $releaseDir)
    $scriptContent = $scriptContent.Replace("{{GROQ_MODEL}}", $escapedGroqModel)
    Set-Content -Path $issPath -Value $scriptContent -Encoding utf8

    $installerName = "AI-ADS-Studio-Setup-Windows-{0}" -f (Get-Date -Format "yyyyMMdd-HHmmss")
    & $innoCompiler "/F$installerName" $issPath
    if ($LASTEXITCODE -ne 0) {
        throw "Inno Setup failed to create the installer."
    }

    $installerPath = Join-Path $releaseDir "$installerName.exe"
    if (-not (Test-Path $installerPath)) {
        throw "Inno Setup completed without producing the expected installer."
    }
    Write-Host "Windows installer created: $installerPath"
}
finally {
    Remove-Item -LiteralPath $stageDir -Recurse -Force -ErrorAction SilentlyContinue
}
