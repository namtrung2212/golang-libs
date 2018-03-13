package GRPC

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	Const "iparking/share/const"
	Logger "iparking/share/libs/logger"
	Bytes "iparking/share/utils/bytes"
	"sync"
	"time"

	context "golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type GRPCConnection struct {
	Name       string
	Address    string
	Connection *grpc.ClientConn
	Lock       sync.RWMutex
	Context    context.Context
	CreatedAt  int64
}

func (this *GRPCConnection) Close() {

	this.Lock.Lock()
	defer this.Lock.Unlock()

	if this.Connection != nil {
		this.Connection.Close()
	}
}

type GRPCClientConfig struct {
	ClientCert string
	ClientKey  string
	ServerCert string
	MaxConn    int
}

type GRPCClient struct {
	Config      *GRPCClientConfig
	DialOptions []grpc.DialOption
	Connections map[string]*GRPCConnection
	Lock        sync.RWMutex
}

func (this *GRPCClient) Reset(config *GRPCClientConfig) error {

	//-------------- TLS ----------------------
	certificate, err := tls.LoadX509KeyPair(config.ClientCert, config.ClientKey)
	if err != nil {
		return err
	}

	bs, err := ioutil.ReadFile(config.ServerCert)
	if err != nil {
		return err
	}

	certPool := x509.NewCertPool()
	if ok := certPool.AppendCertsFromPEM(bs); !ok {
		return Const.ErrDB_FailedAppendPEM
	}

	transportCreds := credentials.NewTLS(&tls.Config{
		MinVersion:               tls.VersionTLS12,
		CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		},
		InsecureSkipVerify: true,
		Certificates:       []tls.Certificate{certificate},
		RootCAs:            certPool,
	})

	this.DialOptions = []grpc.DialOption{
		grpc.WithTransportCredentials(transportCreds),
	}

	this.Config = config
	this.CloseAll()

	return nil

}

func (this *GRPCClient) GetConnection(srvName string) *GRPCConnection {

	this.Lock.Lock()
	defer this.Lock.Unlock()

	if this.Connections == nil {
		this.Connections = make(map[string]*GRPCConnection)
	}

	conn, _ := this.Connections[srvName]
	return conn

}

func (this *GRPCClient) Connect(name string, addr string) *GRPCConnection {

	this.Lock.RLock()

	if this.Connections == nil {
		this.Connections = make(map[string]*GRPCConnection)
	}

	if conn, ok := this.Connections[name]; ok {
		defer this.Lock.RUnlock()
		return conn
	}
	this.Lock.RUnlock()

	this.Lock.Lock()
	defer this.Lock.Unlock()
	if conn, ok := this.Connections[name]; ok {
		return conn
	}

	conn, err := grpc.DialContext(context.Background(), addr, this.DialOptions...)
	if err != nil {
		return nil
	}

	grpcConnection := &GRPCConnection{
		Name:       name,
		Address:    addr,
		Connection: conn,
		Context:    context.Background(),
		CreatedAt:  time.Now().UnixNano(),
	}
	this.Connections[name] = grpcConnection

	go func(c *GRPCClient, con *GRPCConnection) {

		select {
		case <-con.Context.Done():

			c.Lock.Lock()
			defer c.Lock.Unlock()

			delete(c.Connections, con.Name)

			if err := con.Connection.Close(); err != nil {
				Logger.WriteLog("Try closing connection to " + con.Name + " at address: " + con.Address + " with error : " + err.Error())
			}

			if err := con.Context.Err(); err != nil {
				Logger.WriteLog("Context attached to " + con.Name + " at address: " + con.Address + " with error : " + err.Error())
			}
			return
		}
	}(this, grpcConnection)

	return grpcConnection
}

func (this *GRPCClient) ConnectAll(services map[string]string) {

	olds, news := make(map[string]bool), make(map[string]bool)
	for _, v := range this.Connections {
		olds[v.Name] = true
	}
	for name, _ := range services {
		news[name] = true
	}
	if news == nil {
		news = make(map[string]bool)
	}

	// close none-existed clients
	if olds != nil && len(olds) > 0 {

		for _, v := range this.Connections {

			_, okOld := olds[v.Name]
			_, okNew := news[v.Name]

			if okOld && !okNew {
				v.Close()
			}
		}

	}

	// connect new clients
	for name, addr := range services {
		this.Connect(name, addr)
	}

}

func (this *GRPCClient) CloseAll() {

	if this.Connections == nil {
		return
	}

	for _, v := range this.Connections {
		v.Close()
	}

	this.Connections = make(map[string]*GRPCConnection)
}

func (this *BaseRequest) LoadParams(params interface{}) {

	this.Params, _ = Bytes.Encode(params)

}

func (this *GRPCClient) Call(srvName string, req BaseRequest, result interface{}) error {

	conn := this.GetConnection(srvName)
	conn.Lock.RLock()
	defer conn.Lock.RUnlock()

	req.ReqAt = time.Now().UnixNano()

	session := NewGRPCServiceClient(conn.Connection)
	res, err := session.Execute(context.Background(), &req)

	if err != nil {
		return err
	}

	Bytes.Decode(res.Result, &result)
	return nil
}
