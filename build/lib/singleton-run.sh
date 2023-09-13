cmdDir=$(dirname "$0")
source $cmdDir/lib.sh

cmdDir=$(dirname $(ireadlink "$0"))

command="$1"

chmod +x $cmdDir/tcpt

function Pid() {
    ps -ef | grep tcpt.script | grep -v 'grep' | awk -F ' ' '{print $2}' | head
}

if [ "start" == "$command" ]; then

    pid=$(Pid)
    if [ "$pid" != "" ]; then
        echo "is already running ($pid)"
    else
        nohup $cmdDir/tcpt script $cmdDir/tcpt.script > /dev/null 2>&1 &
        sleep 1
        if [ "$(Pid)" != "" ]; then
            echo success
        else
            echo faild
        fi
    fi

elif [ "stop" == "$command" ]; then

    pid=$(Pid)
    if [ "$pid" != "" ]; then
        kill "$pid"
        echo has been killed "$pid"
    else
        echo is not running
    fi

    # ps -ef | grep tcpt.script | grep -v 'grep' | awk -F ' ' '{system("kill "$2); print "killed "$2}'
    
elif [ "status" == "$command" ]; then

    pid=$(Pid)
    if [ "$pid" != "" ]; then
        echo "is running ($pid)"
    else
        echo is not running
    fi

fi

