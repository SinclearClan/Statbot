$env:GOOS = "linux"
$env:GOARCH = "arm"
$env:GOARM = "7"
go build -o Statbot_pi2 main.go
