function ireadlink(){

    which readlink > /dev/null
    if [ $? -eq 0 ]; then
        echo `readlink -f "$0"`
        return 0
    fi

    which greadlink > /dev/null
    if [ $? -eq 0 ]; then
        echo `greadlink -f "$0"`
        return 0
    fi

    return 1
    # if [ "$(ismacos)" == "yes" ]; then
    #     # echo `greadlink -f "$0"`
    #     echo `readlink -f "$0"`
    # else
    #     echo `readlink -f "$0"`
    # fi
}
