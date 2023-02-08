# 介绍

关于 golang debug 和 profiling 的仓库。

## 1. 目录说明

```bash
.
├── benchmark       # 如何写性能测试
├── docs            # 保存文档、图片
├── profiling       # profiling 简介
└── memory          # 内存相关 topic
    ├── escape      # 内存逃逸
    ├── gc          # golang gc 简介
    └── iface       # interface
```

## 2. 大纲

### 2.1 内存相关

1. [golang 编译器内存逃逸分析](./memory/escape/README.md)
2. [interface{} 相关介绍](./memory/iface/README.md)
3. [golang gc 相关介绍](./memory/gc/README.md)

### 2.2 性能剖析

1. [golang 性能剖析介绍](./profiling/README.md)

### 2.3 性能测试

1. [如何写性能测试](./benchmark/README.md)

## TODO

- [x] 如何写性能测试
- [x] golang gc 介绍
