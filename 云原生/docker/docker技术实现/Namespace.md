#namespace
# Namespace
## namespace常用用法
>查看当前系统的namespace
```
lsns -t <type>
```
>查看某进程的namespace
```
ls -la /proc/<pid>/ns/
```
>进入某namespace运行命令
```
nsnter -t <pid> -n ip addr
```
