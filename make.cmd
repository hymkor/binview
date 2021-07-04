@echo off
setlocal
set "PROMPT=[make] "
call :"%~1" "%~2"
endlocal
exit /b

:""
    go build -ldflags "-s -w"
    exit /b

:"package"
    for %%I in ("%CD%") do set "NAME=%%~nI"
    if "%~1" == "" (
        set /P "VERSION=Version ? "
    ) else (
        set "VERSION=%1"
    )
    for %%I in (386 amd64) do (
        set GOARCH=%%I
        call :""
        zip -9 %NAME%-%VERSION%-windows-%%I.zip %NAME%.exe
    )
    set GOARCH=amd64
    set GOOS=linux
    call :""
    tar zcvf %NAME%-%VERSION%-linux-amd64.zip %NAME%
    exit /b
