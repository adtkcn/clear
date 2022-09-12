具体使用步骤：

GO111MODULE=on；
执行命令 go mod init 在当前目录下生成一个 go.mod 文件

执行 go mod tidy 命令，它会添加缺失的模块以及移除不需要的模块。执行后会生成 go.sum 文件(模块下载条目)。添加参数-v，例如 go mod tidy -v 可以将执行的信息，即删除和添加的包打印到命令行；

go mod vendor 生成 vendor 文件夹，将依赖导入到项目下

第一步 : 编译 go build -ldflags "-s -w"
其中 -ldflags 里的 -s 去掉符号信息， -w 去掉 DWARF 调试信息，得到的程序就不能用 gdb 调试了

压缩输出的文件： ./upx .\clear.exe
