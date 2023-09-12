package main

import (
	"fmt"
	"os"
	"strings"

	"tcp-tunnel/lan"
	"tcp-tunnel/logger"
	"tcp-tunnel/wan"

	"github.com/yymmiinngg/goargs"
)

func main() {

	// 这里可以替换成 `os.Args` 以处理控制台命令行
	// flag.Parse()
	var argsArr = os.Args
	// 模板
	template := `
    Usage: {{COMMAND}} <TYPE> {{OPTION}}
    运行一个TCP隧道，TYPE(类型): { LAN, WAN }
	
	# LAN              处于子网，通常无公网IP的局域网服务器
	# WAN              处于公网，具备公网IP的互联网服务器

    ? -h, --help     # 显示帮助后退出
    ? -v, --version  # 显示版本后退出
`

	// 定义变量
	var type_ string

	// 编译模板
	args, err := goargs.Compile(template)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	// 绑定变量
	args.StringOperan("TYPE", &type_, "")

	// 处理参数
	err = args.Parse(argsArr, goargs.AllowUnknowOption)

	// 显示帮助
	if args.HasItem("-H", "--help") {
		fmt.Println("--------------------------------------------------")
		fmt.Println(args.Usage())
	}

	// 显示版本
	if args.HasItem("--version") {
		fmt.Println("--------------------------------------------------")
		fmt.Println("v0.0.1")
	}

	// 错误输出
	if err != nil {
		fmt.Println(err.Error())
		fmt.Println("--------------------------------------------------")
		fmt.Println(args.Usage())
		return
	}

	logger.Type = type_

	if strings.ToLower(type_) == "lan" {
		lan.Start()
	} else if strings.ToLower(type_) == "wan" {
		wan.Start()
	} else {
		fmt.Printf("Unknow type %s\n", type_)
		fmt.Println("--------------------------------------------------")
		fmt.Println(args.Usage())
	}
}
