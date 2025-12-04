
## 服务器性能分析

## pprof 端口：
`robot: 20004`  
`login: 21004`     
`gate:  22004`     
`lobby:  23004`  
`actor:  24004`   
`bill:   28004`  
`idip:   29004`  

## 实时可视化性能指标
可实时查看运行在linux系统的服务性能指标,
不兼容windows系统
`http://ip:port/debug/statsviz/`

## Golang标准pprof指标
#### 安装 Graphviz
WebDev:  
`常用软件\后端\windows_10_cmake_Release_graphviz-install-6.0.2-win64.exe`  

#### ActorServer为例查看内存、CPU等信息

查看pprof
http://localhost:24004/debug/pprof

allocs：查看过去所有内存分配的样本。  
block：查看导致阻塞同步的堆栈跟踪。  
cmdline： 当前程序的命令行的完整调用路径。  
goroutine：查看当前所有运行的 goroutines 堆栈跟踪。  
heap：查看活动对象的内存分配情况。  
mutex：查看导致互斥锁的竞争持有者的堆栈跟踪。  
profile： 默认进行 30s 的 CPU Profiling，得到一个分析用的 profile 文件。  
threadcreate：查看创建新 OS 线程的堆栈跟踪。  
trace：查看当前程序执行的痕迹。  

#### 查看内存  
`go tool pprof http://localhost:24004/debug/pprof/heap`  

#### 查看CPU性能  
`go tool pprof http://localhost:24004/debug/pprof/profile`  
或者  
`go tool pprof -http="localhost:8081" http://localhost:24004/debug/pprof/profile`  

#### 查看goroutine  
`go tool pprof http://localhost:24004/debug/pprof/goroutine`  


#### 机器人Robot查看内存、CPU等信息
`go tool pprof http://localhost:20004/debug/pprof/heap`

`go tool pprof http://localhost:20004/debug/pprof/goroutine`  
