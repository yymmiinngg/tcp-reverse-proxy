package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"tcp-tunnel/lan"
	"tcp-tunnel/wan"

	"github.com/yymmiinngg/goargs"
)

func main() {

	// 这里可以替换成 `os.Args` 以处理控制台命令行
	// flag.Parse()
	var argsArr = os.Args
	// 模板
	template := `
	Usage: {{COMMAND}} <MODE> [SCRIPT-FILE] {{OPTION}}
	
	# MODE: { LAN, WAN, SCRIPT }
	
	#   LAN     处于子网，通常无公网IP的局域网服务器
	#   WAN     处于公网，具备公网IP的互联网服务器
	#   SCRIPT  加载脚本文件，脚本文件中可设置多行参数

	# SCRIPT-FILE:
	
	#   脚本文件中可配置多个隧道连接，格式如下：

	#   WAN -a 0.0.0.0:8081 -s 0.0.0.0:9981
	#   WAN -a 0.0.0.0:8082 -s 0.0.0.0:9982
	#   LAN -a 127.0.0.1:8081 -s 127.0.0.1:9981
	#   LAN -a 127.0.0.1:8082 -s 127.0.0.1:9982

    ? -h, --help     # 显示帮助后退出
    ? -v, --version  # 显示版本后退出

	本程序用于将局域网端口反向映射到公网，连接链路如下：
    -------------------------------------------+-------------------------------------
    lan                                        | wan
    -------------------------------------------+-------------------------------------
    内网应用地址             服务端地址        |    转发端口         公网应用端口
    application-address <--> server-address <- | -> server-port <--> application-port
    -------------------------------------------+-------------------------------------

	`

	// 定义变量
	var mode_ string
	var script_ string

	// 编译模板
	args, err := goargs.Compile(template)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	// 绑定变量
	args.StringOperan("MODE", &mode_, "")
	args.StringOperan("SCRIPT-FILE", &script_, "")

	// 处理参数
	err = args.Parse(argsArr, goargs.AllowUnknowOption)

	// 显示版本
	if args.HasItem("-v", "--version") {
		fmt.Println("v0.0.1")
		return
	}

	// 显示帮助
	if args.HasItem("-h", "--help") && (mode_ == "" || !strings.Contains(" LAN | WAN ", strings.ToUpper(mode_))) {
		fmt.Println(args.Usage())
		return
	}

	// 错误输出
	if err != nil {
		fmt.Println(err.Error())
		fmt.Println(args.Usage())
		os.Exit(1)
	}

	var start = func(mode string, argsarr []string) {
		if strings.ToLower(mode) == "lan" {
			lan.Start(argsarr)
		} else if strings.ToLower(mode) == "wan" {
			wan.Start(argsarr)
		} else {
			fmt.Printf("Unknow mode %s\n", mode)
			os.Exit(1)
		}
	}

	if strings.ToLower(mode_) == "script" {
		if script_ == "" {
			fmt.Println("Need argument SCRIPT-FILE in SCRIPT mode")
			os.Exit(1)
		}
		cmdLines, err := readLines(script_)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		for _, cmdLine := range cmdLines {
			if strings.TrimSpace(cmdLine) == "" {
				continue
			}
			cmds := strings.Split(cmdLine, " ")
			go start(cmds[0], cmds)
		}
		time.Sleep(1000 * time.Hour)

	} else {
		start(mode_, os.Args)
	}
}

func readLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, strings.TrimSpace(scanner.Text()))
	}
	return lines, scanner.Err()
}
