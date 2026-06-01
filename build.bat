@echo off
rem Сборка IpDrop как GUI-приложения (без чёрного окна консоли).
go build -ldflags "-H windowsgui" -o ipdrop.exe .
if %errorlevel%==0 (
  echo Готово: ipdrop.exe
) else (
  echo Ошибка сборки.
)
