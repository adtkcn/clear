xcopy  .\static /S /Y .\bin\static
@REM # 1 目标平台的体系架构（386、amd64、arm） 
set GOARCH=amd64
@REM #2 目标平台的操作系统（darwin、freebsd、linux、windows）
set GOOS=linux
@REM #3 编译 使用-o指定你要生成的文件名称，勿需指定可以去掉（参考：go build main.go）
go build -ldflags "-s -w"  -o ./bin/clear-linux
.\upx.exe ./bin/clear-linux

@REM 打包window
set GOOS=windows
go build -ldflags "-s -w" -o ./bin/clear-windows.exe
.\upx.exe ./bin/clear-windows.exe

@REM 打包苹果darwin
set GOOS=darwin
go build -ldflags "-s -w" -o ./bin/clear-darwin
.\upx.exe ./bin/clear-darwin
