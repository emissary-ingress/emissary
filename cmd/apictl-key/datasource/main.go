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

type Datasource interface {
	Close() error
	AddDomain(string, string, string, string, string) error
	DomainExists(string) (bool, error)
}

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

func (d *PostgresqlDatasource) Close() error {
	return d.db.Close()
}

func (d *PostgresqlDatasource) AddDomain(domainName string, ip string, hostname string, requesterContact string, requesterIp string) error {
	if ip != "" {
		return d.addIpDomain(domainName, ip, requesterContact, requesterIp)
	}
	if hostname != "" {
		return d.addHostnameDomain(domainName, hostname, requesterContact, requesterIp)
	}
	return fmt.Errorf("cannot add aes_domains entry without ip_address or hostname")
}

func (d *PostgresqlDatasource) addIpDomain(domainName string, ip string, requesterContact string, requesterIp string) error {
	stmt, err := d.db.Prepare("INSERT INTO aes_domains(domain, ip_address, requester_ip_address, requester_contact) VALUES($1, $2, $3, $4)")
	if err != nil {
		return err
	}
	_, err = stmt.Exec(domainName, ip, requesterIp, requesterContact)
	if err != nil {
		return err
	}
	return nil
}

func (d *PostgresqlDatasource) addHostnameDomain(domainName string, hostname string, requesterContact string, requesterIp string) error {
	stmt, err := d.db.Prepare("INSERT INTO aes_domains(domain, hostname, requester_ip_address, requester_contact) VALUES($1, $2, $3, $4)")
	if err != nil {
		return err
	}
	_, err = stmt.Exec(domainName, hostname, requesterIp, requesterContact)
	if err != nil {
		return err
	}
	return nil
}

func (d *PostgresqlDatasource) DomainExists(domainName string) (bool, error) {
	var exists bool
	err := d.db.
		QueryRow("SELECT exists (SELECT domain FROM aes_domains WHERE domain=$1);", domainName).
		Scan(&exists)
	if err != nil && err != sql.ErrNoRows {
		return false, err
	}
	return exists, nil
}
