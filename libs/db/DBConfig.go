package DB

type DBConfig struct {
	Type                    string
	DB                      string
	Username                string
	Password                string
	Master                  []string
	Slaves                  []string
	CaCert                  string
	ClientCert              string
	ClientKey               string
	Tls                     string
	Args                    string
	MaxIdleConn             int
	MaxOpenConn             int
	ConnMaxLifetimeInMinute int
}
