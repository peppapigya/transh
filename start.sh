#!/bin/bash

: '
  启动回收站工具脚本
  go版本1.25
'

# 检查go是否安装

check_go() {
  if ! command -v go &> /dev/null; then
    echo "未安装go环境"
    return 1
  fi
  return 0
}

# 安装依赖
install_dep(){
  echo "正在安装所需依赖......"
  if [ ! -e "./go.mod" ]; then
    echo "不存在go.mod文件"
    go mod init
  fi
  go mod download
  echo "Done!"
}

main() {
  echo "回收站启动......"
  if ! check_go; then
    exit 1
  fi

  install_dep

  go build -o transh .

  if $? != 0; then
    echo "编译失败"
    exit 1
  fi

  go run main.go

  echo "回收站启动成功"

}