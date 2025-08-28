package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	trashDir     = ".local/share/Trash"
	filesDirName = "files"
	infoDirName  = "info"
	infoFileExt  = ".trashinfo"
)

type TrashInfo struct {
	Path         string
	DeletionDate string
}

func getTrashDirs() (string, string, string) {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("错误: 无法获取用户主目录: %v\n", err)
		os.Exit(1)
	}

	baseDir := filepath.Join(home, trashDir)
	filesDir := filepath.Join(baseDir, filesDirName)
	infoDir := filepath.Join(baseDir, infoDirName)

	return baseDir, filesDir, infoDir
}

func ensureTrashDirs() {
	baseDir, filesDir, infoDir := getTrashDirs()

	for _, dir := range []string{baseDir, filesDir, infoDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Printf("错误: 无法创建目录 %s: %v\n", dir, err)
			os.Exit(1)
		}
	}
}

func putToTrash(paths []string) {
	_, filesDir, infoDir := getTrashDirs()

	for _, path := range paths {
		absPath, err := filepath.Abs(path)
		if err != nil {
			fmt.Printf("错误: 无法获取绝对路径 %s: %v\n", path, err)
			continue
		}

		if _, err := os.Stat(absPath); os.IsNotExist(err) {
			fmt.Printf("警告: %s 不存在，跳过\n", absPath)
			continue
		}

		// 生成唯一文件名
		name := filepath.Base(absPath)
		timestamp := time.Now().Format("20060102150405")
		uniqueName := fmt.Sprintf("%s.%s", name, timestamp)

		// 移动文件到回收站
		trashPath := filepath.Join(filesDir, uniqueName)
		if err := os.Rename(absPath, trashPath); err != nil {
			fmt.Printf("错误: 无法移动 %s 到回收站: %v\n", absPath, err)
			continue
		}

		// 创建信息文件
		infoFile := filepath.Join(infoDir, uniqueName+infoFileExt)
		infoContent := fmt.Sprintf("[Trash Info]\nPath=%s\nDeletionDate=%s\n",
			absPath, time.Now().Format("2006-01-02T15:04:05"))

		if err := ioutil.WriteFile(infoFile, []byte(infoContent), 0644); err != nil {
			fmt.Printf("错误: 无法创建信息文件: %v\n", err)
			// 尝试恢复文件
			os.Rename(trashPath, absPath)
			continue
		}

		fmt.Printf("已移动到回收站: %s\n", absPath)
	}
}

func listTrash() {
	_, _, infoDir := getTrashDirs()

	files, err := ioutil.ReadDir(infoDir)
	if err != nil {
		fmt.Printf("错误: 无法读取回收站信息: %v\n", err)
		return
	}

	if len(files) == 0 {
		fmt.Println("回收站为空")
		return
	}

	fmt.Println("回收站内容:")
	fmt.Printf("%-4s %-30s %-50s %-20s\n", "序号", "文件名", "原路径", "删除时间")
	fmt.Println(strings.Repeat("-", 110))

	for i, file := range files {
		if filepath.Ext(file.Name()) != infoFileExt {
			continue
		}

		infoFile := filepath.Join(infoDir, file.Name())
		content, err := ioutil.ReadFile(infoFile)
		if err != nil {
			fmt.Printf("错误: 无法读取信息文件 %s: %v\n", file.Name(), err)
			continue
		}

		// 解析信息文件
		var originalPath, deletionDate string
		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "Path=") {
				originalPath = strings.TrimPrefix(line, "Path=")
			} else if strings.HasPrefix(line, "DeletionDate=") {
				deletionDate = strings.TrimPrefix(line, "DeletionDate=")
			}
		}

		trashName := strings.TrimSuffix(file.Name(), infoFileExt)
		fmt.Printf("%-4d %-30s %-50s %-20s\n", i+1, trashName, originalPath, deletionDate)
	}
}

func restoreFromTrash(names []string) {
	_, filesDir, infoDir := getTrashDirs()

	for _, name := range names {
		infoFile := filepath.Join(infoDir, name+infoFileExt)
		trashFile := filepath.Join(filesDir, name)

		if _, err := os.Stat(infoFile); os.IsNotExist(err) {
			fmt.Printf("错误: %s 不在回收站中\n", name)
			continue
		}

		// 读取信息文件获取原始路径
		content, err := ioutil.ReadFile(infoFile)
		if err != nil {
			fmt.Printf("错误: 无法读取信息文件: %v\n", err)
			continue
		}

		var originalPath string
		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "Path=") {
				originalPath = strings.TrimPrefix(line, "Path=")
				break
			}
		}

		if originalPath == "" {
			fmt.Printf("错误: 信息文件损坏: %s\n", infoFile)
			continue
		}

		// 确保目标目录存在
		originalDir := filepath.Dir(originalPath)
		if err := os.MkdirAll(originalDir, 0755); err != nil {
			fmt.Printf("错误: 无法创建目录 %s: %v\n", originalDir, err)
			continue
		}

		// 恢复文件
		if err := os.Rename(trashFile, originalPath); err != nil {
			fmt.Printf("错误: 无法恢复文件: %v\n", err)
			continue
		}

		// 删除信息文件
		os.Remove(infoFile)
		fmt.Printf("已恢复: %s\n", originalPath)
	}
}

func emptyTrash() {
	_, filesDir, infoDir := getTrashDirs()

	files, err := ioutil.ReadDir(infoDir)
	if err != nil {
		fmt.Printf("错误: 无法读取回收站信息: %v\n", err)
		return
	}

	if len(files) == 0 {
		fmt.Println("回收站已经是空的")
		return
	}

	fmt.Printf("即将清空回收站，共 %d 个项目\n", len(files))
	fmt.Print("确认清空回收站？(y/N): ")

	reader := bufio.NewReader(os.Stdin)
	confirm, _ := reader.ReadString('\n')
	confirm = strings.TrimSpace(confirm)

	if confirm == "y" || confirm == "Y" {
		os.RemoveAll(filesDir)
		os.RemoveAll(infoDir)
		ensureTrashDirs() // 重新创建空目录
		fmt.Println("回收站已清空")
	} else {
		fmt.Println("操作已取消")
	}
}

func removeFromTrash(names []string) {
	_, filesDir, infoDir := getTrashDirs()

	for _, name := range names {
		infoFile := filepath.Join(infoDir, name+infoFileExt)
		trashFile := filepath.Join(filesDir, name)

		if _, err := os.Stat(infoFile); os.IsNotExist(err) {
			fmt.Printf("错误: %s 不在回收站中\n", name)
			continue
		}

		// 删除文件和信息文件
		if err := os.RemoveAll(trashFile); err != nil {
			fmt.Printf("错误: 无法删除文件: %v\n", err)
			continue
		}

		if err := os.Remove(infoFile); err != nil {
			fmt.Printf("错误: 无法删除信息文件: %v\n", err)
			continue
		}

		fmt.Printf("已永久删除: %s\n", name)
	}
}

func showUsage() {
	fmt.Println("回收站管理工具")
	fmt.Println("用法:")
	fmt.Println("  trash put <文件/目录>    - 将文件/目录移动到回收站")
	fmt.Println("  trash list              - 列出回收站中的文件")
	fmt.Println("  trash restore <文件名>  - 从回收站恢复文件")
	fmt.Println("  trash empty             - 清空回收站")
	fmt.Println("  trash rm <文件名>       - 删除回收站中的特定文件")
}

func main() {
	ensureTrashDirs()

	if len(os.Args) < 2 {
		showUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	args := os.Args[2:]

	switch command {
	case "put":
		if len(args) == 0 {
			fmt.Println("错误: 请指定要删除的文件或目录")
			os.Exit(1)
		}
		putToTrash(args)
	case "list":
		listTrash()
	case "restore":
		if len(args) == 0 {
			fmt.Println("错误: 请指定要恢复的文件名")
			os.Exit(1)
		}
		restoreFromTrash(args)
	case "empty":
		emptyTrash()
	case "rm":
		if len(args) == 0 {
			fmt.Println("错误: 请指定要删除的文件名")
			os.Exit(1)
		}
		removeFromTrash(args)
	default:
		showUsage()
		os.Exit(1)
	}
}
