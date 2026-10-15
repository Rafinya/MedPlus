@echo off
setlocal
cd /d "%~dp0\backend"
if "%~1" neq "" (
    set "PORT=%~1"
)
echo ========================================================
echo   Starting MedPlus POS Backend Server
echo   Port: %PORT% (Default from .env or 8082)
echo   Docs: http://localhost:%PORT%/docs.html
echo   App:  http://localhost:%PORT%/
echo ========================================================
go run .
pause
