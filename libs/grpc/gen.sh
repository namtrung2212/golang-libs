my_dir=`dirname $0`
protoc $my_dir/GRPCBase.proto --go_out=plugins=grpc:$my_dir