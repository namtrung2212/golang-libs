package DB

import (
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"fmt"
	"io/ioutil"
	Const "iparking/share/const"
	"strings"

	"github.com/go-sql-driver/mysql"
	"github.com/linxGnu/mssqlx"
)

type DBInstance struct {
	db *mssqlx.DBs
}

func (this *DBInstance) Configure(c *DBConfig) error {

	if c.Type != "mysql" {
		return Const.ErrDB_MysqlOnly
	}

	if len(c.Tls) > 0 {
		if strings.HasSuffix(c.Args, "&") {
			c.Args += "tls=" + c.Tls
		} else {
			c.Args += "&tls=" + c.Tls
		}

		rootCertPool := x509.NewCertPool()
		pem, err := ioutil.ReadFile(c.CaCert)
		if err != nil {
			return err
		}
		if ok := rootCertPool.AppendCertsFromPEM(pem); !ok {
			return Const.ErrDB_FailedAppendPEM
		}

		// clientCert := make([]tls.Certificate, 0, 1)
		// certs, err := tls.LoadX509KeyPair(c.ClientCert, c.ClientKey)
		// if err != nil {
		// 	return err
		// }
		// clientCert = append(clientCert, certs)

		mysql.RegisterTLSConfig(c.Tls, &tls.Config{
			RootCAs: rootCertPool,
			// Certificates:             clientCert,
			MinVersion:               tls.VersionTLS12,
			CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
			PreferServerCipherSuites: false,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
				tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_RSA_WITH_AES_256_CBC_SHA,
			},
			InsecureSkipVerify: true,
		})
	}

	if c.Master != nil && len(c.Master) > 0 {
		for i := range c.Master {
			c.Master[i] = fmt.Sprintf("%s:%s@(%s)/%s?%s", c.Username, c.Password, c.Master[i], c.DB, c.Args)
		}
	}

	if c.Slaves != nil && len(c.Slaves) > 0 {
		for i := range c.Slaves {
			c.Slaves[i] = fmt.Sprintf("%s:%s@(%s)/%s?%s", c.Username, c.Password, c.Slaves[i], c.DB, c.Args)
		}
	}

	return nil
}

func (this *DBInstance) Connect(c *DBConfig) error {

	if err := this.Configure(c); err != nil {
		return err
	}

	_db, errors := mssqlx.ConnectMasterSlaves(c.Type, c.Master, c.Slaves)
	if _db == nil {
		return fmt.Errorf("Connection to DB Fail %v", errors)
	}

	if this.db != nil {
		this.db.Destroy()
	}

	this.db = _db
	this.db.SetMaxIdleConns(c.MaxIdleConn)
	this.db.SetMaxOpenConns(c.MaxOpenConn)

	return nil
}

func (this *DBInstance) Begin() (*sql.Tx, error) {
	return this.db.Begin()
}

func (this *DBInstance) Exec(query string, args ...interface{}) (sql.Result, error) {
	return this.db.Exec(query, args...)
}

func (this *DBInstance) Select(dest interface{}, query string, args ...interface{}) error {
	return this.db.Select(dest, query, args...)
}

func (this *DBInstance) Get(dest interface{}, query string, args ...interface{}) error {
	return this.db.Get(dest, query, args...)
}

func (this *DBInstance) Instance() *mssqlx.DBs {
	return this.db
}
