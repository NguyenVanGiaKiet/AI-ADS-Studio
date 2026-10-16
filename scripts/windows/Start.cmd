@echo off
setlocal
cd /d "%~dp0"

if not exist ".env" (
  echo Application configuration is missing. Repair or reinstall the application.
  pause
  exit /b 1
)
if not exist "runtime\python.exe" (
  echo The bundled Python runtime is missing. Repair or reinstall the application.
  pause
  exit /b 1
)
if not exist "ffmpeg\ffmpeg.exe" (
  echo FFmpeg is missing. Repair or reinstall the application.
  pause
  exit /b 1
)
if not exist "ffmpeg\ffprobe.exe" (
  echo FFprobe is missing. Repair or reinstall the application.
  pause
  exit /b 1
)

set "PATH=%CD%\runtime;%CD%\ffmpeg;%PATH%"
set "PIPER_PYTHON=%CD%\runtime\python.exe"
start "AI ADS Studio" /min "%~dp0AI-ADS-Studio.exe"
powershell -NoProfile -ExecutionPolicy Bypass -Command "$ready=$false; for($i=0; $i -lt 40; $i++){ try { Invoke-WebRequest -UseBasicParsing http://127.0.0.1:8080/api/health -TimeoutSec 1 | Out-Null; $ready=$true; break } catch { Start-Sleep -Milliseconds 500 } }; if($ready){ Start-Process 'http://127.0.0.1:8080' } else { Write-Error 'AI ADS Studio did not start. Check the server window.'; exit 1 }"
if errorlevel 1 pause
