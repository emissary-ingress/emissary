package datasource

import (
	"database/sql"
	"fmt"
	"net/url"

	"github.com/jackc/pgx"
	"github.com/jackc/pgx/log/logrusadapter"
	"github.com/jackc/pgx/stdlib"
	"github.com/sirupsen/logrus"
)

type PostgresqlDatasource struct {
	l  *logrus.Logger
	db *sql.DB
}

type DomainEntry struct {
	DomainName       string
	IP               string
	Hostname         string
	EdgectlInstallId string
	AESInstallId     string
	RequesterContact string
	RequesterIp      string
}

type Datasource interface {
	Close() error
	AddDomain(DomainEntry) error
	DomainNameExists(string) (bool, error)
}

// NewDatasource initializes a new SQL datasource connection
func NewDatasource(logger *logrus.Logger, pgURL *url.URL) (*PostgresqlDatasource, error) {
	connConfig, err := pgx.ParseURI(pgURL.String())
	if err != nil {
		return nil, err
	}

	connConfig.Logger = logrusadapter.NewLogger(logger)
	db := stdlib.OpenDB(connConfig)

	if err := validateSchema(db); err != nil {
		return nil, err
	}

	return &PostgresqlDatasource{
		l:  logger,
		db: db,
	}, nil
}

// Close closes the SQL datasource connection
func (d *PostgresqlDatasource) Close() error {
	return d.db.Close()
}

// AddDomain will insert a new aes_domain row in the SQL datasource
func (d *PostgresqlDatasource) AddDomain(e DomainEntry) error {
	if e.IP == "" && e.Hostname == "" {
		return fmt.Errorf("cannot add aes_domains entry without ip_address or hostname")
	}
	stmt, err := d.db.Prepare("INSERT INTO aes_domains(domain, ip_address, hostname, edgectl_install_id, aes_install_id, requester_ip_address, requester_contact) VALUES($1, $2, $3, $4, $5, $6)")
	if err != nil {
		return err
	}
	_, err = stmt.Exec(e.DomainName, nullString(e.IP), nullString(e.Hostname), nullString(e.EdgectlInstallId), nullString(e.AESInstallId), e.RequesterIp, e.RequesterContact)
	if err != nil {
		return err
	}
	return nil
}

// DomainNameExists validates a domain name already exists in database
func (d *PostgresqlDatasource) DomainNameExists(domainName string) (bool, error) {
	var exists bool
	err := d.db.
		QueryRow("SELECT exists (SELECT domain FROM aes_domains WHERE domain=$1);", domainName).
		Scan(&exists)
	if err != nil && err != sql.ErrNoRows {
		return false, err
	}
	return exists, nil
}

func nullString(s string) sql.NullString {
	if len(s) == 0 {
		return sql.NullString{}
	}
	return sql.NullString{
		String: s,
		Valid:  true,
	}
}
