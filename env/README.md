# env

轻量级 .env 配置文件管理工具，零依赖。

## 安装

```bash
go get github.com/1szt/go-pkg/env
```

## 使用

在项目入口（如 main）初始化一次路径，之后任何包 import 该包即可直接读取：

```go
package main

import "github.com/1szt/go-pkg/env"

func main() {
	// 0. 设置配置文件路径（只在入口调用一次；不调用则默认 data/.env）
	env.Path("data/config.env")

	// 1. 声明并自动兜底落盘（想加几行注释随便加）
	env.Add("LFDD_PORT", "9081", "注释1", "注释2", "注释...")
	// 文件内容（首次运行自动生成，重复声明不覆盖）：
	// # 注释1
	// # 注释2
	// LFDD_PORT = 9081

	// 2. 正常读取（环境变量优先，文件兜底，不写盘）
	port := env.Get("LFDD_PORT")

	// 3. 强行只读本地配置文件
	localPort := env.GetFile("LFDD_PORT")

	// 4. 强行只读环境变量
	envPort := env.GetEnv("LFDD_PORT")

	// 5. 修改配置值并落盘（只改值，静态注释雷打不动）
	_ = env.Set("LFDD_PORT", "9090")

	_, _, _ = port, localPort, envPort
}
```

其他业务包无需任何初始化，直接 import 即可：

```go
package service

import "github.com/1szt/go-pkg/env"

func Start() {
	port := env.Get("LFDD_PORT") // 读到的就是入口 Path 指定的文件
	_ = port
}
```

## API

| 函数 | 说明 |
| --- | --- |
| `Path(p string)` | 设置配置文件路径，入口调用一次；不调用默认 `data/.env` |
| `Add(key, value, comments...)` | 声明配置项，文件中不存在时自动追加注释+默认值 |
| `Get(key)` | 读取：系统环境变量优先，文件兜底 |
| `GetFile(key)` | 只读配置文件 |
| `GetEnv(key)` | 只读系统环境变量 |
| `Set(key, value)` | 修改值并落盘，注释保持不动 |

## 特性

- 零依赖
- `Add()` 自动生成带注释的 .env 文件，重复声明不覆盖
- 每次 `Get()` 直接读文件，天然热加载
- 并发不安全（配置几乎不改动，无影响）
