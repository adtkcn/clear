@REM # 1 目标平台的体系架构（386、amd64、arm） 
set GOARCH=amd64
@REM #2 目标平台的操作系统（darwin、freebsd、linux、windows）
set GOOS=linux
wails3 task linux:build

@REM 打包window
set GOOS=windows
wails3 task windows:build

@REM 打包苹果darwin
set GOOS=darwin
wails3 task darwin:build
