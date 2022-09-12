


1. 执行 go mod tidy 命令，它会添加缺失的模块以及移除不需要的模块。执行后会生成 go.sum 文件(模块下载条目)。
2. 添加参数-v，例如 go mod tidy -v 可以将执行的信息，即删除和添加的包打印到命令行；

3. go mod vendor 生成 vendor 文件夹，将依赖导入到项目下

```bash
go mod tidy #移除不需要的模块
go get
go build -ldflags "-s -w" #其中 -ldflags 里的 -s 去掉符号信息， -w 去掉 DWARF 调试信息，不能用 gdb 调试了
./upx.exe ./clear.exe #压缩文件
./clear.exe #运行
```
