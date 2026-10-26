@echo off
setlocal
cd /d "%~dp0"

where ffmpeg >nul 2>nul
if errorlevel 1 (
  echo FFmpeg is required. Install FFmpeg and make sure ffmpeg.exe and ffprobe.exe are on PATH.
  echo Then run Setup.cmd again.
  pause
  exit /b 1
)
where ffprobe >nul 2>nul
if errorlevel 1 (
  echo ffprobe.exe is required. Install FFmpeg and make sure ffmpeg.exe and ffprobe.exe are on PATH.
  echo Then run Setup.cmd again.
  pause
  exit /b 1
)

if not exist ".venv\Scripts\python.exe" (
  where py >nul 2>nul
  if not errorlevel 1 (
    py -3 -m venv .venv
  ) else (
    where python >nul 2>nul
    if errorlevel 1 (
      echo Python 3 is required. Install Python and run Setup.cmd again.
      pause
      exit /b 1
    )
    python -m venv .venv
  )
  if errorlevel 1 (
    echo Could not create the Python environment.
    pause
    exit /b 1
  )
)

".venv\Scripts\python.exe" -m pip install -r requirements.txt
if errorlevel 1 (
  echo Could not install required Python packages.
  pause
  exit /b 1
)

if not exist ".env" copy ".env.example" ".env" >nul
echo.
echo Setup complete. Configure your Gemini API key in .env.
notepad ".env"
pause
