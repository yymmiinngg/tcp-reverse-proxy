package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"tcp-tunnel/client"
	"tcp-tunnel/lan"
	"tcp-tunnel/logger"
	"tcp-tunnel/wan"

	"github.com/yymmiinngg/goargs"
)

// Name Name
var Name string = "tcpt"

// Version Version
var Version string = "dev"

// Platform Platform
var Platform string = "unknow"

// BuildTime BuildTime
var BuildTime string = "unknow"

// GoVersion GoVersion
var GoVersion string = "unknow"

func main() {

	var argsArr = os.Args
	// 模板
	template := `
	Usage: {{COMMAND}} <MODE> [SCRIPT-FILE] {{OPTION}}
	
	# MODE: { LAN, WAN, SCRIPT }
	
	#   LAN     Run a LAN client to forward traffic from WAN to the application port
	#   WAN     Run a WAN server to forward traffic from user clients to LAN client
	#   SCRIPT  Load a script file to run multiple LAN or WAN side programs.

	# SCRIPT-FILE:
	
	#   Script file content like (Multiple line)：

	#   WAN -s :9981
	#   WAN -s :9982
	#   LAN -a 10.0.0.1:8081 -s 100.100.100.1:9981 -o :8081 
	#   LAN -a 10.0.0.1:8082 -s 100.100.100.1:9982 -o :8082

	+ -l, --logger   # Output log to:
	#                    - console: Out to console (Default)
	#                    - User specified file, like: /var/log/tcprp-out.log
	? -d, --debug    # Output debug message, There are a lot of logs in debug mode

    ?     --help     # Show Help and Exit
    ?     --version  # Show Version and Exit
	`

	// 定义变量
	var mode_ string
	var script_ string
	var logger_ string
	var debug bool

	// 编译模板
	args, err := goargs.Compile(template)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	// 绑定变量
	args.StringOperan("MODE", &mode_, "")
	args.StringOperan("SCRIPT-FILE", &script_, "")
	args.StringOption("-l", &logger_, "console")
	args.BoolOption("-d", &debug, false)

	// 处理参数
	err = args.Parse(argsArr, goargs.AllowUnknowOption)

	// 显示版本
	if args.Has("--version", false) {
		fmt.Println("Version  :", Name, Version)
		fmt.Println("BuildTime:", BuildTime)
		fmt.Println("Platform :", Platform)
		fmt.Println("GoVersion:", GoVersion)
		return
	}

	// 显示帮助
	if args.Has("--help", false) && (mode_ == "" || !strings.Contains(" LAN | WAN | CLIENT | SCRIPT ", strings.ToUpper(mode_))) {
		fmt.Println(args.Usage())
		return
	}

	// 错误输出
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	// 创建日志对象
	log, err := logger.MakeLogger(mode_, logger_, debug)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	// 开始函数
	var start = func(mode string, argsarr []string) {
		if strings.ToLower(mode) == "lan" {
			lan.Start(argsarr, log)
		} else if strings.ToLower(mode) == "wan" {
			wan.Start(argsarr, log)
		} else if strings.ToLower(mode) == "client" {
			client.Start(argsarr, log)
		} else if strings.ToLower(mode) == "script" {
			fmt.Printf("Can't run script mode in script file\n")
			os.Exit(1)
			return
		} else {
			fmt.Printf("Unknow mode %s\n", mode)
			os.Exit(1)
			return
		}
	}

	// 运行指令
	if strings.ToLower(mode_) == "script" { // 运行 Script
		if script_ == "" {
			fmt.Println("In SCRIPT mode, the parameter SCRIPT-FILE is mandatory.")
			os.Exit(1)
			return
		}
		cmdLines, err := readLines(script_)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
			return
		}
		for _, cmdLine := range cmdLines {
			if strings.TrimSpace(cmdLine) == "" {
				continue
			}
			cmds := strings.Split(cmdLine, " ")
			go start(cmds[0], cmds)
		}
		// 100年不退出
		time.Sleep(365.25 * 24 * 100 * time.Hour)
	} else { // 运行 Wan | Lan
		start(mode_, os.Args)
	}

}

// 读脚本文件
func readLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.Index(line, "#") == 0 {
			continue
		}
		lines = append(lines, line)
	}
	return lines, scanner.Err()
}
