package GRPC

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	Const "iparking/share/const"
	bytes "iparking/share/utils/bytes"
	"log"
	"net"
	"reflect"
	"time"

	context "golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type GRPCServerConfig struct {
	ServerPort string
	ServerKey  string
	ServerCert string
	ClientCert string
}

type GRPCService interface {
	GetRequestObject(method string) interface{}
}

type GRPCServer struct {
	Config   *GRPCServerConfig
	Server   *grpc.Server
	Listener *net.Listener
	Services map[string]GRPCService
}

func (this *GRPCServer) loadClientCA(clientCA string) (*x509.CertPool, error) {

	bs, err := ioutil.ReadFile(clientCA)
	if err != nil {
		return nil, err
	}

	certPool := x509.NewCertPool()
	if ok := certPool.AppendCertsFromPEM(bs); !ok {
		log.Fatal("failed to append client certs")
	}

	return certPool, nil
}

func (this *GRPCServer) Configure(config *GRPCServerConfig) error {

	this.Services = make(map[string]GRPCService)
	this.Config = config

	serverCA, err := tls.LoadX509KeyPair(config.ServerCert, config.ServerKey)
	if err != nil {
		return err
	}

	clientCA, err := this.loadClientCA(config.ClientCert)
	if err != nil {
		return err
	}

	opts := []grpc.ServerOption{
		grpc.Creds(credentials.NewTLS(&tls.Config{
			MinVersion:               tls.VersionTLS12,
			CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
			PreferServerCipherSuites: true,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
				tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_RSA_WITH_AES_256_CBC_SHA,
			},
			ClientAuth:   tls.RequireAndVerifyClientCert,
			Certificates: []tls.Certificate{serverCA},
			ClientCAs:    clientCA,
		})),
	}

	this.Server = grpc.NewServer(opts...)

	RegisterGRPCServiceServer(this.Server, this)

	return nil
}

func (this *GRPCServer) Start(port string) error {

	if this.Listener != nil {
		log.Printf("stoping listener : %v", (*this.Listener).Addr())
		(*this.Listener).Close()
	}

	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("failed to listen: %s", err)
		return err
	}

	this.Listener = &listener
	log.Printf("start listener : %v", (*this.Listener).Addr())

	if err := this.Server.Serve(listener); err != nil {
		log.Fatalf("failed to serve: %v", err)
		return err
	}

	return nil
}

func (this *GRPCServer) Register(service GRPCService, name string) {
	this.Services[name] = service
}

func (this *GRPCServer) Execute(ctx context.Context, req *BaseRequest) (*BaseResponse, error) {

	log.Printf("req: %v", req)

	if req.Service == "" || this.Services[req.Service] == nil {
		return &BaseResponse{
			Error: Const.ErrServiceNotAvailable.Error(),
			ResAt: time.Now().UnixNano(),
		}, Const.ErrServiceNotAvailable
	}

	service := this.Services[req.Service]

	method := reflect.ValueOf(service).MethodByName(req.Method)

	reqObj := service.GetRequestObject(req.Method)
	reqObjPtr := reflect.New(reflect.TypeOf(reqObj)).Interface()
	if err := bytes.Decode(req.Params, &reqObjPtr); err != nil {
		log.Printf("err: %v", err)
		return nil, err
	}

	params := []reflect.Value{reflect.ValueOf(reqObjPtr).Elem()}

	response := method.Call(params)

	result := response[0].Interface()
	err := response[1].Interface()

	// log.Printf("result: %v", result)
	// log.Printf("err: %v", err)
	if err != nil {
		return &BaseResponse{
			Error: fmt.Errorf("%v", err).Error(),
			ResAt: time.Now().UnixNano(),
		}, fmt.Errorf("%v", err)
	}

	encoded, err := bytes.Encode(result)
	if err != nil {
		return &BaseResponse{
			Error: fmt.Errorf("%v", err).Error(),
			ResAt: time.Now().UnixNano(),
		}, fmt.Errorf("%v", err)
	}

	return &BaseResponse{
		Result: encoded,
		Error:  "",
		ResAt:  time.Now().UnixNano(),
	}, nil
}
