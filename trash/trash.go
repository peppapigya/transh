package trash

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"transh/utils"

	"github.com/olekukonko/tablewriter"
)

const (
	// trashDir 回收站目录
	trashDir = ".local/share/Transh"
	// transhFilesDir 实际存放文件的目录
	transhFilesDir = "fileInfos"
	// trashLogDir 日志目录，用户数据回滚操作
	trashLogDir = "logs"
	// defaultTrashBackupDir TRASH_BACKUP_DIR 默认备份目录
	defaultTrashBackupDir = ".local/share/backup"
	// 回收网站备份信息后缀
	trashBackupSuffix = "backup"
)

// FileLogInfo 文件日志信息，主要是方便提取文件信息
type FileLogInfo struct {
	// 开始信息，主要包括时间
	BeginInfo string `json:""`
	FileName  string `json:"file_name"`
	// 原始文件路径
	OriginPath string `json:"origin_path"`
	// 目标文件路径
	TargetPath string `json:"target_path"`
	// 操作人
	Operator user.User `json:"operator"`
	// 文件大小
	FileSize int64 `json:"file_size"`
	// 删除时间
	DeletionDate string `json:"deletion_date"`
}

// 获取所有回收站的目录
func getAllTranshDir() []string {
	dir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("获取用户家目录失败：%v\n", err)
		os.Exit(1)
	}
	baseDir := filepath.Join(dir, trashDir)
	return []string{
		baseDir,                                   // 回收站目录
		filepath.Join(baseDir, transhFilesDir),    // 实际存放文件的目录
		filepath.Join(baseDir, trashLogDir),       // 日志目录
		filepath.Join(dir, defaultTrashBackupDir), // 默认备份目录
	}
}

// Usage 使用方法
func Usage() {
	fmt.Println("Linux简易回收站工具")
	fmt.Println("作者:peppa-pig")
	fmt.Println("用法:")
	fmt.Println("  transh -p <文件> <文件>... ：将多个指定的文件放入回收站")
	fmt.Println("  transh -l 或者 transh --list : 列出回收站中的文件信息")
	fmt.Println("  transh -c 或者 transh --clear : 清空回收站")
	fmt.Println("  transh -r <文件> 或者 transh --restore : 从回收站中恢复指定文件")
	fmt.Println("  transh -d <文件> 或者 transh --delete : 删除回收站中的指定文件")
	fmt.Println("  transh -b <路径> 或者 transh --backup <路径> : 备份回收站到指定路径（可选）")
	fmt.Println("  transh -h 或者 transh --help :获取帮助信息")
}

// CreateTranshDir 检查回收站是否存在并创建对应目录
func CreateTranshDir() {
	// 获取用户根目录
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("获取用户家目录失败：%v\n", err)
		os.Exit(1)
	}
	baseDir := filepath.Join(userHomeDir, trashDir)
	transhFilesDir := filepath.Join(baseDir, transhFilesDir)
	trashLogDir := filepath.Join(baseDir, trashLogDir)
	// 创建目录
	for _, dir := range []string{baseDir, transhFilesDir, trashLogDir} {
		// todo 暂定权限为0755
		if err := os.MkdirAll(dir, 0755); os.IsNotExist(err) {
			fmt.Printf("创建目录失败 %s: %v\n", dir, err)
			os.Exit(1)
		}
	}
}

// PutFileToTransh 将文件放入回收站
func PutFileToTransh(args []string) {
	if checkArgsIsEmpty(args) {
		fmt.Printf("需要输入指定文件或目录\n")
		os.Exit(1)
	}
	transhDirs := getAllTranshDir()

	fmt.Println(args)
	fmt.Printf("是否确认将以上文件放入回收？(y/n):")
	reader := bufio.NewReader(os.Stdin)
	confirm, _ := reader.ReadString('\n')
	confirm = strings.TrimSpace(confirm)
	if confirm != "y" && confirm != "Y" {
		fmt.Println("操作已取消")
		os.Exit(0)
	}
	for _, file := range args {
		// 保存文件信息
		newFilePath, oldFilePath, fileInfo := saveFileInfoToDisk(file, transhDirs[1])
		// 记录文件的日志信息，todo后续加上目录之后需要加上消息队列去异步写日志，减少用户等待时间
		saveLogInfoToDisk(newFilePath, oldFilePath, transhDirs[2], fileInfo)
	}
	fmt.Printf("文件放入回收站成功\n")
}

// GetTrashFileList 获取回收站列表文件,返回文件列表
// todo 1.后续加上缓存以及根据文件名称搜索,2. 自定义排序,3.增加分页功能,4.按照时间去排序
func GetTrashFileList(fileName string) {
	allTrashDirs := getAllTranshDir()
	logDir := allTrashDirs[2]
	// 获取所有文件信息
	files, err := os.ReadDir(logDir)
	if err != nil {
		fmt.Printf("错误：获取回收站文件列表失败：%v\n", err)
		os.Exit(1)
	}
	var fileInfos []FileLogInfo
	for _, file := range files {
		// 读取文件最后一行信息
		if file.IsDir() {
			continue
		}
		// 获取文件路径信息，todo 这里可能会有问题
		fileAbsPath := filepath.Join(logDir, file.Name())
		lastFileLogInfo := readLastLineFromFile(fileAbsPath)
		fileInfos = append(fileInfos, lastFileLogInfo)
	}

	printRecycleBin(fileInfos)
}

// ClearTranshFileInfo 清空回收站
func ClearTranshFileInfo() {
	allTranshDirs := getAllTranshDir()

	fmt.Printf("是否清空回收站(y/n):")
	reader := bufio.NewReader(os.Stdin)
	confirm, _ := reader.ReadString('\n')
	confirm = strings.TrimSpace(confirm)
	if confirm != "y" && confirm != "Y" {
		fmt.Printf("操作已取消\n")
		os.Exit(0)
	}
	fmt.Printf("正在清空回收站...\n")
	// 将回收站的所有文件压缩到~/.local/share/backup
	gzipAllTranshFile(allTranshDirs[0], allTranshDirs[3])
	// 删除回收站文件
	removeDirAllFiles(allTranshDirs[1], "回收站")
	removeDirAllFiles(allTranshDirs[2], "日志文件")
}

// RestoreTranshFile 恢复文件，目前只支持文件不能指定目录
func RestoreTranshFile(fileNames []string) {
	transhDirs := getAllTranshDir()
	// 遍历回收站的内容
	for _, file := range fileNames {
		baseFileName := filepath.Base(file)
		logFilePath := filepath.Join(transhDirs[2], baseFileName)
		// 检查回收站会否有相应的文件
		if _, err := os.Stat(logFilePath); err != nil {
			fmt.Printf("错误：回收站没有对应的文件，文件：{%s},错误内容：{%v}", file, err)
			os.Exit(1)
		}
	}
}

// DeleteTranshFile 删除回收站文件
func DeleteTranshFile(fileNames []string) {}

// BackupTranshFile 定期备份回收站文件
func BackupTranshFile(backupDir []string) {}

// ===================================== 辅助方法 ================================

// 压缩回收站文件，默认是tar.gz
// @param dir 回收站根目录
// @param targetDir 目标目录
// todo dxg: 后续指定压缩形式、压缩文件以后的文件大小、目录
func gzipAllTranshFile(dir string, targetDir string) {
	//getFileIfo(dir)

	if ok := os.MkdirAll(targetDir, 0755); ok != nil {
		fmt.Printf("错误：创建目录{%v}失败：%v\n", targetDir, ok)
		os.Exit(1)
	}
	timestamp := time.Now().Format("20250829173530")
	fileName := fmt.Sprintf("transh-back-%v.tar.gz", timestamp)

	command := exec.Command("tar", "-czvf", filepath.Join(targetDir, fileName), "-C", dir, ".")
	fmt.Printf("压缩命令：tar -czvf %v -C  %v .\n", filepath.Join(targetDir, fileName), dir)
	if err := command.Run(); err != nil {
		fmt.Printf("错误：压缩文件失败：%v\n", err)
		os.Exit(1)
	}
	fmt.Printf("文件压缩成功，文件路径：%s\n", filepath.Join(targetDir, fileName))
}

// 删除指定目录下所有的文件，
// todo 待优化,错误日志写到一个统一的日志目录
func removeDirAllFiles(dir string, message string) {
	files, err := os.ReadDir(dir)
	if err != nil {
		fmt.Printf("错误：读取目录失败：%v\n", err)
		os.Exit(1)
	}
	successCnt, failCnt := 0, 0
	for _, file := range files {
		path := filepath.Join(dir, file.Name())
		if err := os.RemoveAll(path); err != nil {
			fmt.Printf("错误：删除文件失败：%v\n", err)
			failCnt++
			continue
		}
		successCnt++
	}
	fmt.Printf("删除 %s 文件成功，成功删除文件数：%d   , 失败文件数：%d\n", message, successCnt, failCnt)
}

func printRecycleBin(files []FileLogInfo) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"序号", "文件名称", "文件大小(Byte)", "删除时间", "操作人", "原始路径", "目标路径"})
	var totalSize int64
	for i, f := range files {
		table.Append([]string{
			fmt.Sprintf("%d", i+1),
			f.FileName,
			fmt.Sprintf("%d", f.FileSize),
			f.DeletionDate,
			f.Operator.Username,
			f.OriginPath,
			f.TargetPath,
		})
		totalSize += f.FileSize
	}
	table.SetFooter([]string{"", "", "", "", "",
		fmt.Sprintf("文件总数量: %d", len(files)),
		fmt.Sprintf("文件总大小: %s", strconv.FormatInt(totalSize, 10))})
	table.Render()
}

// 判断参数是否为空
func checkArgsIsEmpty(args []string) bool {
	return len(args) == 0
}

/**
 * 保存文件信息
 * @param file 待删除的文件
 * @param transInfoDir 保存文件信息的目录
 * @return 保存成功返回新的文件path和原来的文件文件path
 */
func saveFileInfoToDisk(file string, transInfoDir string) (string, string, os.FileInfo) {
	oldFileAbs := getFileAbs(file)
	fmt.Printf("正在将文件 %s 放入回收站...\n", oldFileAbs)
	// 判断文件是否存在
	fileInfo := getFileIfo(oldFileAbs)

	timestamp := time.Now().Format("20250827151130")
	newFileName := fmt.Sprintf("%s.%s", fileInfo.Name(), timestamp)
	newFilePath := filepath.Join(transInfoDir, newFileName)
	// 将文件写入到回收站目录
	if err := os.Rename(oldFileAbs, newFilePath); err != nil {
		fmt.Printf("错误：移动文件 %s 进入回收站失败：%v\n", oldFileAbs, err)
		os.Exit(1)
	}
	return newFilePath, oldFileAbs, fileInfo
}

/**
 * 保存日志信息
 * @param newFilePath 新文件路径
 * @param logDir 日志保存目录
 */
func saveLogInfoToDisk(newFilePath string, oldFilePath string, logDir string, fileInfo os.FileInfo) {
	fmt.Printf("正在保存日志信息...\n")
	logFileName := fmt.Sprintf("%s.%s", filepath.Base(oldFilePath), trashBackupSuffix)

	currentUser := getUserInfo()
	logFileInfo := &FileLogInfo{
		FileName:     filepath.Base(oldFilePath),
		BeginInfo:    fmt.Sprintf("[ %s ]", time.Now().Format("2006-01-02 15:04:05")),
		DeletionDate: time.Now().Format("2006-01-02 15:04:05"),
		FileSize:     fileInfo.Size(),
		OriginPath:   oldFilePath,
		TargetPath:   newFilePath,
		Operator:     *currentUser,
	}
	logFileContent := utils.ParserToJson(logFileInfo)
	logFilePath := filepath.Join(logDir, logFileName)
	// 以追加的形式写入日志文件 todo 后续增加超时重试
	file, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("错误:打开日志文件失败: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		err := file.Close()
		if err != nil {
			fmt.Printf("错误:关闭日志文件失败: %v\n", err)
			return
		}
	}()
	if _, err := file.Write(append([]byte(logFileContent), '\n')); err != nil {
		fmt.Printf("错误:写入日志文件失败: %v\n", err)
		// 尝试恢复文件
		os.Rename(newFilePath, oldFilePath)
		os.Exit(1)
	}
}

// 获取文件信息
func getFileIfo(filePath string) os.FileInfo {
	fileInfo, err := os.Stat(filePath)
	// 文件不存在直接退出todo  后续加上回滚之前的数据保证原子性
	if err != nil {
		fmt.Printf("错误:文件 %s 不存在: %v\n", filePath, err)
		os.Exit(1)
	}
	// 如果删除的是个目录的话直接返回，
	if fileInfo.IsDir() {
		fmt.Printf("错误：%s 该文件是个目录，不是一个文件", filePath)
		os.Exit(1)
	}
	return fileInfo
}

/**
 * 获取当前用户信息
 *
 * @return 当前用户信息
 */
func getUserInfo() *user.User {
	currentUser, err := user.Current()
	if err != nil {
		fmt.Printf("获取用户信息失败: %v\n", err)
		os.Exit(1)
	}
	return currentUser
}

/*
 * 获取文件绝对路径
 * @param file 文件路径
 * @return 文件绝对路径
 */
func getFileAbs(file string) string {
	fileAbs, err := filepath.Abs(file)
	if err != nil {
		fmt.Printf("错误:获取文件 %s 绝对路径失败: %v\n", file, err)
		os.Exit(1)
	}
	return fileAbs
}

// 读取文件最后一行
// @param file为绝对路径
func readLastLineFromFile(file string) FileLogInfo {
	cmd := exec.Command("tail", "-n", "-1", file)
	if cmd == nil {
		fmt.Printf("错误：创建 tail 命令失败，请重试")
		os.Exit(1)
	}
	stdout, err := cmd.Output()
	args := cmd.Args
	if err != nil {
		command := "tail "
		for _, arg := range args {
			command += string(arg) + " "
		}
		fmt.Printf("错误：执行 %s 命令失败，请重试!{%v}\n", command, err)
		os.Exit(1)
	}
	outInfo := strings.TrimSpace(string(stdout))
	if outInfo == "" {
		fmt.Printf("警告：日志文件为空\n")
		os.Exit(1)
	}
	var fileLogInfo FileLogInfo
	utils.JsonToStruct(outInfo, &fileLogInfo)
	return fileLogInfo
}
