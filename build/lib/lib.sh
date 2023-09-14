
function ireadlink {

    which readlink > /dev/null
    if [ $? -eq 0 ]; then
        echo `readlink -f "$1"`
        return 0
    fi

    which greadlink > /dev/null
    if [ $? -eq 0 ]; then
        echo `greadlink -f "$1"`
        return 0
    fi

    return 1
}