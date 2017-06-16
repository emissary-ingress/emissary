import sys

import json
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

CONSUMER_TABLE_SQL = '''
CREATE TABLE IF NOT EXISTS consumers (
    consumer_id VARCHAR(64) NOT NULL PRIMARY KEY,
    username VARCHAR(2048) NOT NULL,
    fullname VARCHAR(2048) NOT NULL,
    shortname VARCHAR(2048)
)
'''

CONSUMER_MODULES_TABLE_SQL = '''
CREATE TABLE IF NOT EXISTS consumer_modules (
    consumer_id VARCHAR(64) NOT NULL REFERENCES consumers(consumer_id),
    module_name VARCHAR(2048) NOT NULL,
    module_data JSONB NOT NULL,
    PRIMARY KEY (consumer_id, module_name)
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
            self.cursor.execute(CONSUMER_TABLE_SQL)
            self.cursor.execute(CONSUMER_MODULES_TABLE_SQL)

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
            deleted = self.cursor.rowcount
            self.conn.commit()

            return RichStatus.OK(name=name, deleted=deleted)
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
            deleted = self.cursor.rowcount

            self.conn.commit()

            return RichStatus.OK(name=name, deleted=deleted)
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

    ######## CONSUMER API
    def consumers_where(self, consumer_id=None, username=None):
        if not consumer_id and not username:
            return RichStatus.fromError("one of consumer_id and username is required")

        sql_clauses = []
        hr_items = []
        keys = {}

        if consumer_id:
            sql_clauses.append("(consumer_id = :consumer_id)")
            hr_items.append(consumer_id)
            keys['consumer_id'] = consumer_id

        if username:
            sql_clauses.append("(username = :username)")
            hr_items.append(username)
            keys['username'] = username

        sql = " AND ".join(sql_clauses)
        hr = "/".join(hr_items)

        return RichStatus.OK(sql=sql, hr=hr, keys=keys)

    def fetch_all_consumers(self):
        if not self:
            return self.status

        try:
            self.cursor.execute("SELECT consumer_id, username, fullname, shortname FROM consumers ORDER BY fullname, consumer_id")

            consumers = []

            for consumer_id, username, fullname, shortname in self.cursor:
                consumers.append({ "consumer_id": consumer_id,
                                   "username": username,
                                   "fullname": fullname,
                                   "shortname": shortname })

            return RichStatus.OK(consumers=consumers, count=len(consumers))
        except pg8000.Error as e:
            return RichStatus.fromError("fetch_all_consumers: could not fetch info: %s" % e)

    # Consumers are weird because more than one table is involved in Postgres. We
    # have one table, which we call 'basics' here, with ID and names and that's it.
    # We have another table, which we call 'modules' here, with ID and module info.
    # A given consumer will have exactly one basics entry and zero or more modules
    # entries.
    # 
    # Transactions are important here, since we give users a way to create a
    # consumer with module info already loaded, and we give users a way to separately
    # tweak modules afterward. So we split our API here into _*_basics and _*_modules,
    # and then e.g. fetching a single consumer will use both to DTRT.

    #### fetch

    def _fetch_consumer_basics(self, where):
        if not self:
            return self.status

        try:
            sql = "SELECT consumer_id, username, fullname, shortname FROM consumers WHERE %s" % where.sql

            self.cursor.execute(sql, where.keys)

            if self.cursor.rowcount == 0:
                return RichStatus.fromError("consumer %s not found" % where.hr)

            if self.cursor.rowcount > 1:
                return RichStatus.fromError("consumer %s matched more than one entry?" % where.hr)

            # We know there's exactly one consumer match. Good.

            consumer_id, username, fullname, shortname = self.cursor.fetchone()

            return RichStatus.OK(consumer_id=consumer_id,
                                 username=username,
                                 fullname=fullname,
                                 shortname=shortname)
        except pg8000.Error as e:
            return RichStatus.fromError("fetch_consumer_basics %s: could not fetch info: %s" % (where.hr, e))

    def _fetch_consumer_modules(self, consumer_id, module_name=None):
        if not self:
            return self.status

        try:
            sql = "SELECT module_name, module_data FROM consumer_modules WHERE consumer_id = :consumer_id"

            if module_name:
                sql += " AND module_name = :module_name"

            self.cursor.execute(sql, locals())

            modules = { module_name: module_data 
                        for module_name, module_data in self.cursor }

            return RichStatus.OK(consumer_id=consumer_id, modules=modules)
        except pg8000.Error as e:
            return RichStatus.fromError("fetch_consumer %s: could not fetch info: %s" % (where.hr, e))

    def fetch_consumer(self, consumer_id=None, username=None):
        if not self:
            return self.status

        where = self.consumers_where(consumer_id=consumer_id, username=username)

        if not where:
            return where

        rc = self._fetch_consumer_basics(where=where)

        if not rc:
            return rc

        rc2 = self._fetch_consumer_modules(rc.consumer_id)

        if rc2:
            # XXX Ew hackery.
            rc.info['modules'] = rc2.modules

        return rc

    def fetch_consumer_module(self, consumer_id, module_name=None):
        if not self:
            return self.status

        if not module_name:
            return RichStatus.fromError("fetch_consumer_module: module_name is required")

        return self._fetch_consumer_modules(consumer_id, module_name=module_name)

    #### delete

    def _delete_consumer_basics(self, consumer_id):
        if not self:
            return self.status

        try:
            self.cursor.execute("DELETE FROM consumers WHERE consumer_id = :consumer_id", locals())
            deleted = self.cursor.rowcount

            return RichStatus.OK(consumer_id=consumer_id, deleted=deleted)
        except pg8000.Error as e:
            return RichStatus.fromError("delete_consumer %s: could not delete consumer info: %s" % (consumer_id, e))

    def _delete_consumer_modules(self, consumer_id, module_name=None):
        if not self:
            return self.status

        try:
            sql = "DELETE FROM consumer_modules WHERE consumer_id = :consumer_id"

            if module_name:
                sql += " AND module_name = :module_name"

            self.cursor.execute(sql, locals())
            deleted = self.cursor.rowcount

            return RichStatus.OK(consumer_id=consumer_id, modules_deleted=deleted)
        except pg8000.Error as e:
            what = consumer_id if not module_name else "{} {}".format(consumer_id, module_name)

            return RichStatus.fromError("delete_consumer %s: could not delete consumer module info: %s" % (what, e))

    def delete_consumer(self, consumer_id):
        if not self:
            return self.status

        try:
            # Have to delete modules first since it holds a foreign key
            rc = self._delete_consumer_modules(consumer_id)

            if not rc:
                self.conn.rollback()
                return rc

            modules_deleted = rc.modules_deleted

            rc = self._delete_consumer_basics(consumer_id)

            if not rc:
                self.conn.rollback()
                return rc

            self.conn.commit()
            return RichStatus.OK(consumer_id=consumer_id, deleted=rc.deleted, modules_deleted=modules_deleted)
        except pg8000.Error as e:
            return RichStatus.fromError("delete_consumer %s: could not delete consumer: %s" % (consumer_id, e))

    def delete_consumer_module(self, consumer_id, module_name):
        if not self:
            return self.status

        try:
            rc = self._delete_consumer_modules(consumer_id, module_name=module_name)

            if rc:
                self.conn.commit()
            else:
                self.conn.rollback()
                
            return rc
        except pg8000.Error as e:
            return RichStatus.fromError("delete_consumer_module %s %s: could not delete module: %s" % (consumer_id, module_name, e))

    #### store

    def _store_consumer_basics(self, consumer_id, username, fullname, shortname):
        if not self:
            return self.status

        try:
            self.cursor.execute('''
                INSERT INTO consumers VALUES(:consumer_id, :username, :fullname, :shortname)
                    ON CONFLICT (consumer_id) DO UPDATE SET
                        consumer_id=EXCLUDED.consumer_id, username=EXCLUDED.username, 
                        fullname=EXCLUDED.fullname, shortname=EXCLUDED.shortname
            ''', locals())

            return RichStatus.OK(consumer_id=consumer_id)
        except pg8000.Error as e:
            return RichStatus.fromError("store_consumer %s: could not save info: %s" % (consumer_id, e))

    # Note that we do _one module at a time_ here.
    def _store_consumer_module(self, consumer_id, module_name, module_data_object):
        if not self:
            return self.status

        module_data = json.dumps(module_data_object)

        try:
            self.cursor.execute('''
                INSERT INTO consumer_modules VALUES(:consumer_id, :module_name, :module_data)
                    ON CONFLICT (consumer_id, module_name) DO UPDATE SET
                        consumer_id=EXCLUDED.consumer_id,
                        module_name=EXCLUDED.module_name, module_data=EXCLUDED.module_data
            ''', locals())

            return RichStatus.OK(consumer_id=consumer_id, module_name=module_name)
        except pg8000.Error as e:
            return RichStatus.fromError("store_consumer %s: could not save module info: %s" % (consumer_id, e))

    def store_consumer(self, consumer_id, username, fullname, shortname, modules):
        if not self:
            return self.status

        try:
            rc = self._store_consumer_basics(consumer_id, username, fullname, shortname)

            if not rc:
                self.conn.rollback()
                return rc

            for module_name in modules.keys():
                rc = self._store_consumer_module(consumer_id, module_name, modules[module_name])

                if not rc:
                    self.conn.rollback()
                    return rc

            self.conn.commit()
            return RichStatus.OK(consumer_id=consumer_id)
        except pg8000.Error as e:
            return RichStatus.fromError("store_consumer %s: could not store consumer: %s" % (consumer_id, e))

    def store_consumer_module(self, consumer_id, module_name, module_data_object):
        if not self:
            return self.status

        try:
            rc = self._store_consumer_module(consumer_id, module_name, module_data_object)

            if rc:
                self.conn.commit()
            else:
                self.conn.rollback()

            return rc
        except pg8000.Error as e:
            return RichStatus.fromError("store_consumer_module %s: could not store consumer module: %s" % (consumer_id, e))
