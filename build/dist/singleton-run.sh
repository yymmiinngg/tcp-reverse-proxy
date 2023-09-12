cmdDir=$(dirname "$0")
source $cmdDir/lib.sh

cmdDir=$(dirname $(ireadlink "$0"))

command="$1"


function Pid() {
    ps -ef | grep tcpt.script | grep -v 'grep' | awk -F ' ' '{print $2}' | head
}

if [ "start" == "$command" ]; then

    pid=$(Pid)
    if [ "$pid" != "" ]; then
        echo is running
    else
        nohup $cmdDir/tcpt script $cmdDir/tcpt.script > /dev/null 2>&1 &
    fi

elif [ "stop" == "$command" ]; then

    ps -ef | grep tcpt.script | grep -v 'grep' | awk -F ' ' '{system("kill "$2); print "killed "$2}'
    
elif [ "status" == "$command" ]; then

    ps -ef | grep tcpt.script | grep -v 'grep'
    
fi

