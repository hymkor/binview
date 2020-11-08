@echo off
setlocal
set "PROMPT=[make] "
call :"%1"
endlocal
exit /b

:""
    go build -ldflags "-s -w"
    exit /b

:"package"
    for %%I in ("%CD%") do set "NAME=%%~nI"
    set /P "VERSION=Version ? "
    for %%I in (386 amd64) do (
        set GOARCH=%%I
        call :""
        zip -9 %NAME%-%VERSION%-%%I.zip %NAME%.exe
    )
    exit /b
