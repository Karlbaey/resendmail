alias b := build
alias r := release

build:
    go build .

release:
    go build -ldflags "-s -w"
    upx ./resendmail.exe