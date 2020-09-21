@setlocal
@call :"%1"
@endlocal
@exit /b

:""
    set GOARCH=386
    go build -ldflags "-s -w"
    upx *.exe
    exit /b
