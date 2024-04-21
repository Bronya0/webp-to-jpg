go mod tidy
go build -ldflags="-s -w"  main.go
upx -9 main.exe