// env 轻量级 .env 配置文件管理工具
//
// 用法：
//
//	// 0️⃣ 设置配置文件路径（程序入口调用一次，不设置默认 data/.env）
//	env.Path("data/config.env")
//
//	// 1️⃣ 声明并自动兜底落盘（想加几行注释随便加）
//	env.Add("LFDD_PORT", "9081", "注释1", "注释2", "注释...")
//
//	// 2️⃣ 正常读取（环境变量优先，文件兜底，不写盘）
//	port := env.Get("LFDD_PORT")
//
//	// 3️⃣ 强行只读本地配置文件
//	localPort := env.GetFile("LFDD_PORT")
//
//	// 4️⃣ 强行只读环境变量
//	envPort := env.GetEnv("LFDD_PORT")
//
//	// 5️⃣ 修改配置值并落盘（只改值，静态注释雷打不动）
//	env.Set("LFDD_PORT", "9090")
//
// 特性：
//   - 零依赖
//   - Add() 自动生成带注释的 .env 文件，重复声明不覆盖
//   - 每次 Get() 直接读文件，天然热加载
//   - 并发不安全（配置几乎不改动，无影响）
package env

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// fileName 配置文件路径，通过 Path() 自定义，默认 data/.env
var fileName = "data/.env"

// Path 设置配置文件路径（建议在程序入口只调用一次，之后任意包 Get/Set 均读写该文件）
// 传空字符串则回落到默认路径 data/.env
func Path(p string) {
	p = strings.TrimSpace(p)
	if p == "" {
		p = "data/.env"
	}
	fileName = p
}

// Add 声明一个配置项：文件中不存在该 Key 时自动追加【注释+默认值】，已存在则保持原样
//
// 参数：key 环境变量名，value 默认值，comments 任意多行注释（以 # 写入文件）
func Add(key, value string, comments ...string) {
	// 1. 确保目录存在（目录已存在时 MkdirAll 是空操作）
	if dir := filepath.Dir(fileName); dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Println("创建目录失败:", err)
			return
		}
	}

	// 2. 读取现有内容（文件不存在则为空）
	existingContent, _ := os.ReadFile(fileName)

	// 3. 已存在该 Key 则跳过（逐行行首精确匹配，避免 PORT 被 GRPC_PORT 误判为已存在）
	if fileHasKey(string(existingContent), key) {
		return
	}

	// 4. 注释块 + KEY=VALUE 追加到文件尾部（#注释 \n Key=Value \n\n）
	var comment strings.Builder
	for _, c := range comments {
		comment.WriteString("# ")
		comment.WriteString(c)
		comment.WriteString("\n")
	}
	data := fmt.Sprintf("%s%s=%s\n\n", comment.String(), key, value)

	// 原文件末尾缺换行符时先补一个，避免新行与最后一行黏连被整行注释吞掉
	if len(existingContent) > 0 && existingContent[len(existingContent)-1] != '\n' {
		data = "\n" + data
	}

	f, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("打开/创建文件失败:", err)
		return
	}
	defer f.Close()
	if _, err := f.WriteString(data); err != nil {
		fmt.Printf("追加参数 [%s] 失败: %v\n", key, err)
	}
}

// Get 读取值：系统环境变量优先，其次配置文件；都没有返回空字符串
func Get(key string) string {
	if envVal := os.Getenv(key); envVal != "" {
		return envVal
	}
	return GetFile(key)
}

// GetFile 仅从配置文件读取值，不检查系统环境变量
func GetFile(key string) string {
	content, err := os.ReadFile(fileName)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(content), "\n") {
		if k, v, ok := parseKV(line); ok && k == key {
			return v
		}
	}
	return ""
}

// GetEnv 仅从系统环境变量读取值，不检查配置文件
func GetEnv(key string) string {
	return os.Getenv(key)
}

// Set 修改配置值并落盘：找到 KEY= 只替换值所在行（注释保持不动），没有则追加
func Set(key, value string) error {
	// 1. 确保目录存在（Path 指向任意深层目录时也能落盘）
	if dir := filepath.Dir(fileName); dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	// 2. 读取文件全部内容
	existingContent, err := os.ReadFile(fileName)
	if err != nil {
		// 文件不存在，直接创建新文件写入
		return os.WriteFile(fileName, []byte(key+"="+value+"\n"), 0644)
	}

	lines := strings.Split(string(existingContent), "\n")
	found := false

	// 3. 逐行扫描，替换已有 Key 的值
	for i, line := range lines {
		if k, _, ok := parseKV(line); ok && k == key {
			lines[i] = key + "=" + value
			found = true
			break
		}
	}

	// 4. 没找到则追加
	if !found {
		lines = append(lines, key+"="+value)
	}

	// 5. 写回文件
	return os.WriteFile(fileName, []byte(strings.Join(lines, "\n")), 0644)
}

// parseKV 解析一行 KEY=VALUE：容忍等号两侧空格；注释行/空行/无等号行返回 ok=false
func parseKV(line string) (key, value string, ok bool) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return "", "", false
	}
	i := strings.Index(line, "=")
	if i <= 0 {
		return "", "", false
	}
	return strings.TrimSpace(line[:i]), strings.TrimSpace(line[i+1:]), true
}

// fileHasKey 判断文件中是否已声明该 Key（逐行行首精确匹配，跳过注释行）
func fileHasKey(content, key string) bool {
	for _, line := range strings.Split(content, "\n") {
		if k, _, ok := parseKV(line); ok && k == key {
			return true
		}
	}
	return false
}
