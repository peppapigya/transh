#!/bin/bash

: '
  启动回收站工具脚本
  go版本1.25
'

# 检查go是否安装

check_go() {
  if ! command -v go &> /dev/null; then
    echo "未安装go环境"
    install_go
    return "$?"
  fi
  return 0
}

install_go() {
  echo "尝试安装go环境..."
  cnt=4
  go_url="https://golang.google.cn/dl/go1.25.0.linux-amd64.tar.gz"
  go_install_dir="/usr/local/software/"
  while (( cnt >=0 )) ;do
    if [ ! -e "/tmp/go1.25.0.linux-amd64.tar.gz" ]; then
      wget -O /tmp/go1.25.0.linux-amd64.tar.gz ${go_url}
    fi
    if [ "$?" -eq 0 ]
    then
      echo "从{$go_url},下载go安装包成功,正在安装..."
      # 如果software不存在直接创建目录
       if [ ! -d "${go_install_dir}" ]; then
         mkdir -p ${go_install_dir}
       fi
      sudo rm -rf ${go_install_dir}/go && tar -C ${go_install_dir} -xzf /tmp/go1.25.0.linux-amd64.tar.gz
      echo "export PATH=\$PATH:${go_install_dir}go/bin" >> ~/.bash_profile
      source ~/.bash_profile

      echo "go环境安装成功，go的版本信息：" ${go version}
      return 0
    else
      echo "从{$go_url},下载go安装包失败，正在进行重试..."
    fi
    let cnt--
  done

  echo "go环境安装失败..."
  return 1
}



# 安装依赖
install_dep(){
  echo "正在安装所需依赖......"
  if [ ! -e "./go.mod" ]; then
    echo "不存在go.mod文件"
    go mod init
  fi
  # 设置阿里云镜像避免访问不了github
  go env -w GOPROXY=https://mirrors.aliyun.com/goproxy/,direct
  go mode tidy
  go mod download
  echo "Done!"
}

main() {
  echo "回收站启动......"
  if ! check_go; then
    exit 1
  fi
  cd /usr/local/transh
  install_dep

  go build -o transh .
  if [ "$?" != 0 ];  then
    echo "编译失败"
    exit 1
  fi

  echo "编译成功，将编译好的二进制文件移动到/usr/local/bin ...."
  # 将编译好的文件移动到bin目录下
  sudo mv transh /usr/local/bin
  sudo chmod +x /usr/local/bin/transh

  echo "回收站脚本安装成功"
  echo transh -h
}

main "$@"