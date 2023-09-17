cmdDir=$(dirname "$0")

source $cmdDir/lib.sh

# 获得命令的绝对路径
cmdDir=$(ireadlink "$cmdDir")
if [ "$cmdDir" == "" ]; then
    echo file not found: $cmdDir
    exit 1
fi

# 参数
cmd=tcprp
command="$1"
scriptFileInput="$2"
time=$(date +"%Y%m%d%H%M%S")

# 配置文件
source $cmdDir/tcprps.conf
# 使用默认脚本文件
if [ "$scriptFileInput" == "" ]; then
    scriptFileInput="$defaultScriptFile"
fi

# 获得文件的绝对路径
scriptFile=$(ireadlink "$scriptFileInput")
if [ "$scriptFile" == "" ]; then
    echo file not found: $scriptFileInput
    exit 1
fi

options=""
# 处理配置文件中的参数
if [ "$debug" == "yes" ]; then
    options="$options --debug"
fi

if [ "$logFile" == "" ]; then
    logFile="console"
fi

function Pid {
    ps -ef | grep "$cmdDir/$cmd" | grep "script" | grep "$scriptFile" | grep -v 'grep' | awk -F ' ' '{print $2}' | head
}

function Start {
    pid=$(Pid)
    if [ "$pid" != "" ]; then
        echo "is already running ($pid)"
    else
        chmod +x $cmdDir/$cmd
        if [ "$options" == "" ]; then
            echo $cmdDir/$cmd script "$scriptFile" -l "$logFile"
            nohup $cmdDir/$cmd script "$scriptFile" -l "$logFile" >> $cmdDir/tcprps.console 2>&1 &
        else
            echo $cmdDir/$cmd script "$scriptFile" -l "$logFile" --debug
            nohup $cmdDir/$cmd script "$scriptFile" -l "$logFile" --debug >> $cmdDir/tcprps.console 2>&1 &
        fi
        sleep 1
        if [ "$(Pid)" != "" ]; then
            echo start success
        else
            tail $cmdDir/tcprps.console
            echo start fail
        fi
    fi
}

function Stop {
    pid=$(Pid)
    if [ "$pid" != "" ]; then
        kill "$pid"
        echo has been killed "$pid"
    else
        echo is not running
    fi
}

if [ "start" == "$command" ]; then

    Start

elif [ "stop" == "$command" ]; then

    Stop

elif [ "restart" == "$command" ]; then

    Stop
    Start

elif [ "status" == "$command" ]; then

    pid=$(Pid)
    if [ "$pid" != "" ]; then
        echo "is running ($pid)"
    else
        echo is not running
    fi

else

    printf "Usage: %s <COMMAND> [SCRIPT-FILE]\n" "$(basename $0)"
    printf "    COMMAND: { start | stop | restart | status }\n"
    printf "    The default SCRIPT-FILE is tcprp.script\n"

fi

