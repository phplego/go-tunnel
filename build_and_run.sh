go build

if [[ $? -eq 0 ]]
then
    ./go-tunnel $@
fi

