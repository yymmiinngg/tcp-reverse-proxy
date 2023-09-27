cmdDir=$(dirname "$0")

source $cmdDir/lib.sh

# 获得命令的绝对路径
cmdDir=$(ireadlink "$cmdDir")
if [ "$cmdDir" == "" ]; then
    echo file not found: $cmdDir
    exit 1
fi

# args
cmd=tcprp
command="$1"
scriptFileInput="$2"
time=$(date +"%Y-%m-%d %H:%M:%S")

# config file
source $cmdDir/tcprps.conf

# default script file
if [ "$scriptFileInput" == "" ]; then
    scriptFileInput="$defaultScriptFile"
fi

# absolute file path
scriptFile=$(ireadlink "$scriptFileInput")
if [ "$scriptFile" == "" ]; then
    echo file not found: $scriptFileInput
    exit 1
fi

# options
options=""
if [ "$debug" == "yes" ]; then
    options="-D"
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
        echo "" >> $cmdDir/tcprps.console
        echo "[$time]" >> $cmdDir/tcprps.console
        echo "$cmdDir/$cmd script \"$scriptFile\" -L \"$logFile\" $options" >> $cmdDir/tcprps.console
        d=$(pwd)
        cd $cmdDir
        nohup $cmdDir/$cmd script "$scriptFile" -L "$logFile" $options >> $cmdDir/tcprps.console 2>&1 &
        cd $d
        sleep 1
        if [ "$(Pid)" != "" ]; then
            echo start success >> $cmdDir/tcprps.console
        else
            echo start fail >> $cmdDir/tcprps.console
        fi
        tail $cmdDir/tcprps.console
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

