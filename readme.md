# 一、使用方式
### 1、直接下载打包文件
提供了window、linux、苹果三个amd64架构的版本，但只有windows经过测试
https://github.com/adtkcn/clear/releases

### 2、源码打包方式
```bash
go mod tidy #移除不需要的模块
go get
go build -ldflags "-s -w" #其中-s 去掉符号信息，-w 去掉 DWARF 调试信息，不能用gdb调试
./upx.exe ./clear.exe #压缩文件
./clear.exe #运行或者双击运行
```

# 二、自定义排除和扫描目录
同级目录下新建`config.json`文件
```json5
{
    "white_dir_list": [], // 不扫描的目录，不区分大小写
    "search_dir_list": [] // 待扫描的目录名称，不区分大小写
}
```

## 预览
![预览](./static/page.png)