## golangci-lint 使用文档
使用linter工具对代码进行静态检测，及早发现一些低级错误，统一编码的规范。理论上扫描结果都需要进行处理，如果不处理则需要在对应代码处使用`//nolint`进行手动忽略，并写上忽略原因。

工具官网 https://golangci-lint.run/
> 注：请使用v1版本工具 https://golangci.github.io/legacy-v1-doc/

### linter下载
* 方式1：github release包下载
```
https://github.com/golangci/golangci-lint/releases/download/v1.64.8/golangci-lint-1.64.8-windows-amd64.zip
```
解压并放置到GOPATH/bin目录下。

* 方式2：直接 go install 方式下载
```
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.8
```

### linter使用
* 扫描整个工程，递归所有子目录go文件，下面两个命令等效
```
C:\Users\user\go\src\football\gameserver> golangci-lint run
或者
C:\Users\user\go\src\football\gameserver> golangci-lint run ./..
```

* 扫描指定包，如service包
```
C:\Users\user\go\src\football\gameserver> golangci-lint run .\internal\service\
或者 扫描多个包
C:\Users\user\go\src\football\gameserver> golangci-lint run .\internal\service\ .\internal\model\
```

> 注：一般不单独扫描某个文件