package main

import (
	"os"
	"transh/trash"
)

func main() {
	trash.CreateTranshDir()
	commonds := os.Args
	// 参数小于2时，表示语法错误，需要提示用户用法
	if len(commonds) < 2 {
		trash.Usage()
		os.Exit(1)
	}

	// 获取所有参数
	option := commonds[1]
	args := commonds[2:]
	switch option {
	case "-p", "--put":
		trash.PutFileToTransh(args)
		os.Exit(0)
	case "-l", "--list":
		trash.GetTrashFileList("")
		os.Exit(0)
	case "-c", "--clear":
		trash.ClearTranshFileInfo()
		os.Exit(0)
	case "-r", "restore":
		trash.RestoreTranshFile(args)
		os.Exit(0)
	case "-d", "--delete":
		trash.DeleteTranshFile(args)
		os.Exit(0)
	case "-b", "--backup":
		trash.BackupTranshFile(args)
		os.Exit(0)

	}
}
