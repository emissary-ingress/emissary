import sys

import os

import pg8000

from utils import RichStatus

AMBASSADOR_TABLE_SQL = '''
CREATE TABLE IF NOT EXISTS mappings (
    name VARCHAR(64) NOT NULL PRIMARY KEY,
    prefix VARCHAR(2048) NOT NULL,
    service VARCHAR(2048) NOT NULL,
    rewrite VARCHAR(2048) NOT NULL
)
'''

PRINCIPAL_TABLE_SQL = '''
CREATE TABLE IF NOT EXISTS principals (
    name VARCHAR(64) NOT NULL PRIMARY KEY,
    fingerprint VARCHAR(2048) NOT NULL
)
'''

class AmbassadorStore (object):
    def __init__(self):
        pg8000.paramstyle = 'named'

        self.status = RichStatus.OK()

        # Make sure we have tables and such.
        #
        # All of these functions update self.status if something goes wrong, and they're
        # no-ops if not self.status.

        self.conn = self._get_connection()

        # Get a cursor and verify our database.
        self.cursor = self._get_cursor()
        self._verify_database()

        # Switch autocommit off...
        self._autocommit(False)

        # ...grab a new cursor...
        self.cursor = self._get_cursor()

        # ...and make sure our tables are OK.
        self._verify_tables()

        # At this point we're ready to answer queries...

    def __bool__(self):
        return bool(self.status)

    def __nonzero__(self):
        return bool(self)
        
    def _get_connection(self, autocommit=False):
        # Figure out where the DB lives...

        self.db_name = "postgres"
        self.db_host = "ambassador-store"
        self.db_port = 5432

        if "AMBASSADOR_DB_NAME" in os.environ:
            self.db_name = os.environ["AMBASSADOR_DB_NAME"]

        if "AMBASSADOR_DB_HOST" in os.environ:
            self.db_host = os.environ["AMBASSADOR_DB_HOST"]

        if "AMBASSADOR_DB_PORT" in os.environ:
            self.db_port = int(os.environ["AMBASSADOR_DB_PORT"])

        conn = None

        try:
            conn = pg8000.connect(user="postgres", password="postgres",
                                  database=self.db_name, host=self.db_host, port=self.db_port)

            # Start with autocommit on.
            conn.autocommit = True
        except pg8000.Error as e:
            self.status = RichStatus.fromError("could not connect to database: %s" % e)

        return conn

    def _autocommit(self, setting):
        if not self:
            return

        if self.conn:
            self.conn.autocommit = setting
        else:
            self.status = RichStatus.fromError("cannot set autocommit with no connection")

    def _get_cursor(self):
        if not self:
            return

        cursor = None

        try:
            cursor = self.conn.cursor()
        except pg8000.Error as e:
            self.status = RichStatus.fromError("could not get database cursor: %s" % e)

        return cursor

    def _verify_database(self):
        if not self:
            return

        try:
            self.cursor.execute("SELECT 1 FROM pg_database WHERE datname = 'ambassador'")
            results = self.cursor.fetchall()

            if not results:
                self.cursor.execute("CREATE DATABASE ambassador")
        except pg8000.Error as e:
            self.status = RichStatus.fromError("no ambassador database: %s" % e)

    def _verify_tables(self):
        if not self:
            return

        try:
            self.cursor.execute(AMBASSADOR_TABLE_SQL)
            self.cursor.execute(PRINCIPAL_TABLE_SQL)
            self.conn.commit()
        except pg8000.Error as e:
            self.status = RichStatus.fromError("no data tables: %s" % e)

    ######## MAPPING API
    def fetch_all_mappings(self):
        if not self:
            return self.status

        try:
            self.cursor.execute("SELECT name, prefix, service, rewrite FROM mappings ORDER BY name, prefix")

            mappings = []

            for name, prefix, service, rewrite in self.cursor:
                mappings.append({ 'name': name, 'prefix': prefix, 
                                  'service': service, 'rewrite': rewrite })

            return RichStatus.OK(mappings=mappings, count=len(mappings))
        except pg8000.Error as e:
            return RichStatus.fromError("fetch_all_mappings: could not fetch info: %s" % e)

    def fetch_mapping(self, name):
        if not self:
            return self.status

        try:
            self.cursor.execute("SELECT prefix, service, rewrite FROM mappings WHERE name = :name", locals())

            found = False
            prefix = None
            service = None
            rewrite = None

            for p, s, r in self.cursor:
                found = True
                prefix = p
                service = s
                rewrite = r
                break

            if found:
                return RichStatus.OK(name=name, prefix=prefix, service=service, rewrite=rewrite)
            else:
                return RichStatus.fromError("mapping %s not found" % name)
        except pg8000.Error as e:
            return RichStatus.fromError("fetch_mapping %s: could not fetch info: %s" % (name, e))

    def delete_mapping(self, name):
        if not self:
            return self.status

        try:
            self.cursor.execute("DELETE FROM mappings WHERE name = :name", locals())
            self.conn.commit()

            return RichStatus.OK(name=name)
        except pg8000.Error as e:
            return RichStatus.fromError("delete_mapping %s: could not delete mapping: %s" % (name, e))

    def store_mapping(self, name, prefix, service, rewrite):
        if not self:
            return self.status

        try:
            self.cursor.execute('''
                INSERT INTO mappings VALUES(:name, :prefix, :service, :rewrite)
                    ON CONFLICT (name) DO UPDATE SET
                        name=EXCLUDED.name, prefix=EXCLUDED.prefix, 
                        service=EXCLUDED.service, rewrite=EXCLUDED.rewrite
            ''', locals())
            self.conn.commit()

            return RichStatus.OK(name=name)
        except pg8000.Error as e:
            return RichStatus.fromError("store_mapping %s: could not save info: %s" % (name, e))

    ######## PRINCIPAL API
    def fetch_all_principals(self):
        if not self:
            return self.status

        try:
            self.cursor.execute("SELECT name, fingerprint FROM principals ORDER BY name, fingerprint")

            principals = []

            for name, fingerprint in self.cursor:
                principals.append({ 'name': name, 'fingerprint': fingerprint })

            return RichStatus.OK(principals=principals, count=len(principals))
        except pg8000.Error as e:
            return RichStatus.fromError("fetch_all_principals: could not fetch info: %s" % e)

    def fetch_principal(self, name):
        if not self:
            return self.status

        try:
            self.cursor.execute("SELECT fingerprint FROM principals WHERE name = :name", locals())

            found = False
            fingerprint = None

            for f in self.cursor:
                found = True
                fingerprint = f
                break

            if found:
                return RichStatus.OK(name=name, fingerprint=fingerprint)
            else:
                return RichStatus.fromError("principal %s not found" % name)
        except pg8000.Error as e:
            return RichStatus.fromError("fetch_principal %s: could not fetch info: %s" % (name, e))

    def delete_principal(self, name):
        if not self:
            return self.status

        try:
            self.cursor.execute("DELETE FROM principals WHERE name = :name", locals())
            self.conn.commit()

            return RichStatus.OK(name=name)
        except pg8000.Error as e:
            return RichStatus.fromError("delete_principal %s: could not delete principal: %s" % (name, e))

    def store_principal(self, name, fingerprint):
        if not self:
            return self.status

        try:
            self.cursor.execute('INSERT INTO principals VALUES(:name, :fingerprint)', locals())
            self.conn.commit()

            return RichStatus.OK(name=name)
        except pg8000.Error as e:
            return RichStatus.fromError("store_principal %s: could not save info: %s" % (name, e))
