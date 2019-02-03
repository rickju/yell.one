build_dir=./vendor/build
target=$(build_dir)/bknd-svr
proto_path=.
proto_file=./vnc.proto
proto_go=$(build_dir)/vnc.pb.go

all: _folder $(proto_go) $(target)

_folder:
	mkdir -p ./vendor/build

$(proto_go): $(proto_file)
	protoc -I . --go_out=plugins=grpc:$(build_dir) --proto_path=$(proto_path) $(proto_file)

#go build -gcflags=all="-N -l" $(build_dir)
$(target):
	go build  -o $(target)


#	echo "Compiling for every OS and Platform"
#	------------------------------------------
#	GOOS=linux GOARCH=arm go build -o bin/main-linux-arm main.go
#	GOOS=linux GOARCH=arm64 go build -o bin/main-linux-arm64 main.go
#	GOOS=freebsd GOARCH=386 go build -o bin/main-freebsd-386 main.go
clean:
	rm -f $(build_dir)/*

