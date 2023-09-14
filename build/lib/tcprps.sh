cmdDir=$(dirname "$0")

source $cmdDir/lib.sh
source $cmdDir/tcprps.conf

cmd=tcprp
command="$1"
scriptFileInput="$2"
time=$(date +"%Y%m%d%H%M%S")

# 使用默认脚本文件
if [ "$scriptFileInput" == "" ]; then
    scriptFileInput="$defaultScriptFile"
fi

# 获得文件的实际
scriptFile=$(ireadlink "$scriptFileInput")

if [ "$scriptFile" == "" ]; then
    echo file not found: $scriptFileInput
    exit 1
fi

chmod +x $cmdDir/$cmd

function Pid {
    ps -ef | grep "$scriptFile" | grep -v 'grep' | awk -F ' ' '{print $2}' | head
}

function Start {
    pid=$(Pid)
    if [ "$pid" != "" ]; then
        echo "is already running ($pid)"
    else
        nohup $cmdDir/$cmd script "$scriptFile" -l "$logfile" > .tcprps-console.$time 2>&1 &
        sleep 1
        cat .tcprps-console.$time
        rm -f .tcprps-console.$time
        if [ "$(Pid)" != "" ]; then
            echo script file: $scriptFile
            echo logger file: $logfile
            echo start success
        else
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

