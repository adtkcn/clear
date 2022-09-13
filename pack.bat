xcopy  .\static /S /Y .\bin\clear-linux\static\
xcopy  .\static /S /Y .\bin\clear-windows\static\
xcopy  .\static /S /Y .\bin\clear-darwin\static\

@REM # 1 目标平台的体系架构（386、amd64、arm） 
set GOARCH=amd64
@REM #2 目标平台的操作系统（darwin、freebsd、linux、windows）
set GOOS=linux
go build -ldflags "-s -w"  -o ./bin/clear-linux/clear
.\upx.exe ./bin/clear-linux/clear

@REM 打包window
set GOOS=windows
go build -ldflags "-s -w" -o ./bin/clear-windows/clear.exe
.\upx.exe ./bin/clear-windows/clear.exe

@REM 打包苹果darwin
set GOOS=darwin
go build -ldflags "-s -w" -o ./bin/clear-darwin/clear
.\upx.exe ./bin/clear-darwin/clear
