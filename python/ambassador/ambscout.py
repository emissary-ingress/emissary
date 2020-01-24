from typing import Any, ClassVar, Dict, List, Optional, Tuple, Union
from typing import cast as typecast

import datetime
import json
import logging
import re
import os

import semantic_version

from scout import Scout

# Import version stuff directly from ambassador.VERSION to avoid a circular import.
from .VERSION import Version, Build, BuildInfo

ScoutNotice = Dict[str, str]


class LocalScout:
    def __init__(self, logger, app: str, version: str, install_id: str) -> None:
        self.logger = logger
        self.app = app
        self.version = version
        self.install_id = install_id

        self.events: List[Dict[str, Any]] = []

        self.logger.info(f'LocalScout: initialized for {app} {version}: ID {install_id}')

    def report(self, **kwargs) -> dict:
        self.events.append(kwargs)

        mode = kwargs['mode']
        action = kwargs['action']

        now = datetime.datetime.now().timestamp()

        kwargs['local_scout_timestamp'] = now

        if 'timestamp' not in kwargs:
            kwargs['timestamp'] = now

        self.logger.info(f"LocalScout: mode {mode}, action {action} ({kwargs})")

        return {
            "latest_version": self.version,
            "application": self.app,
            "cached": False,
            "notices": [ { "level": "WARNING", "message": "Using LocalScout, result is faked!" } ],
            "timestamp": now
        }

    def reset_events(self) -> None:
        self.events = []

class AmbScout:
    reTaggedBranch: ClassVar = re.compile(r'^v?(\d+\.\d+\.\d+)(-[a-zA-Z][a-zA-Z]\d+)?$')
    reGitDescription: ClassVar = re.compile(r'-(\d+)-g([0-9a-f]+)$')

    install_id: str
    runtime: str
    namespace: str

    version: str
    semver: Optional[semantic_version.Version]

    _scout: Optional[Union[Scout, LocalScout]]
    _scout_error: Optional[str]

    _notices: Optional[List[ScoutNotice]]
    _last_result: Optional[dict]
    _last_update: Optional[datetime.datetime]
    _update_frequency: datetime.timedelta

    _latest_version: Optional[str] = None
    _latest_semver: Optional[semantic_version.Version] = None

    def __init__(self, install_id=None, update_frequency=datetime.timedelta(hours=12), local_only=False) -> None:
        if not install_id:
            install_id = os.environ.get('AMBASSADOR_CLUSTER_ID',
                                        os.environ.get('AMBASSADOR_SCOUT_ID', "00000000-0000-0000-0000-000000000000"))

        self.install_id = install_id
        self.runtime = "kubernetes" if os.environ.get('KUBERNETES_SERVICE_HOST', None) else "docker"
        self.namespace = os.environ.get('AMBASSADOR_NAMESPACE', 'default')

        self.is_edge_stack = os.path.exists('/ambassador/.edge_stack')
        self.app = "aes" if self.is_edge_stack else "ambassador"
        self.version = Version if self.is_edge_stack else self.parse_git_description(Version, Build)
        self.semver = self.get_semver(self.version)

        self.logger = logging.getLogger("ambassador.scout")
        # self.logger.setLevel(logging.DEBUG)

        self.logger.debug("Ambassador version %s built from %s on %s" % (Version, Build.git.commit, Build.git.branch))
        self.logger.debug("Scout version      %s%s" % (self.version, " - BAD SEMVER" if not self.semver else ""))
        self.logger.debug("Runtime            %s" % self.runtime)
        self.logger.debug("Namespace          %s" % self.namespace)

        self._scout = None
        self._scout_error = None

        self._local_only = local_only

        self._notices = None
        self._last_result = None
        self._update_frequency = update_frequency
        self._latest_version = None
        self._latest_semver = None

        self.reset_cache_time()

    def reset_cache_time(self) -> None:
        self._last_update = datetime.datetime.now() - datetime.timedelta(hours=24)

    def reset_events(self) -> None:
        if self._local_only:
            assert(self._scout)
            typecast(LocalScout, self._scout).reset_events()

    def __str__(self) -> str:
        return ("%s: %s" % ("OK" if self._scout else "??", 
                            self._scout_error if self._scout_error else "OK"))

    @property
    def scout(self) -> Optional[Scout]:
        if not self._scout:
            if self._local_only:
                self._scout = LocalScout(logger=self.logger,
                                         app=self.app, version=self.version, install_id=self.install_id)
                self.logger.debug("LocalScout initialized")
            else:
                try:
                    self._scout = Scout(app=self.app, version=self.version, install_id=self.install_id)
                    self._scout_error = None
                    self.logger.debug("Scout connection established")
                except OSError as e:
                    self._scout = None
                    self._scout_error = str(e)
                    self.logger.debug("Scout connection failed, will retry later: %s" % self._scout_error)

        return self._scout

    def report(self, force_result: Optional[dict]=None, no_cache=False, **kwargs) -> dict:
        _notices: List[ScoutNotice] = []

        # Silly, right?
        use_cache = not no_cache

        env_result = None

        if use_cache:
            env_result = os.environ.get("AMBASSADOR_SCOUT_RESULT", None)

            if env_result:
                force_result = json.loads(env_result)

        result: Optional[dict] = force_result
        result_was_cached: bool = False

        if not result:
            if 'runtime' not in kwargs:
                kwargs['runtime'] = self.runtime

            if 'commit' not in kwargs:
                kwargs['commit'] = Build.git.commit

            if 'branch' not in kwargs:
                kwargs['branch'] = Build.git.branch

            # How long since the last Scout update? If it's been more than an hour,
            # check Scout again.

            now = datetime.datetime.now()

            needs_update = True

            if use_cache:
                if self._last_update:
                    since_last_update = now - typecast(datetime.datetime, self._last_update)
                    needs_update = (since_last_update > self._update_frequency)

            if needs_update:
                if self.scout:
                    result = self.scout.report(**kwargs)

                    self._last_update = now
                    self._last_result = dict(**typecast(dict, result)) if result else None
                else:
                    result = { "scout": "unavailable: %s" % self._scout_error }
                    _notices.append({ "level": "DEBUG",
                                      "message": "scout temporarily unavailable: %s" % self._scout_error })

                # Whether we could talk to Scout or not, update the timestamp so we don't
                # try again too soon.
                result_timestamp = datetime.datetime.now()
            else:
                _notices.append({ "level": "DEBUG", "message": "Returning cached result" })
                result = dict(**typecast(dict, self._last_result)) if self._last_result else None
                result_was_cached = True

                # We can't get here unless self._last_update is set.
                result_timestamp = typecast(datetime.datetime, self._last_update)
        else:
            _notices.append({ "level": "INFO", "message": "Returning forced Scout result" })
            result_timestamp = datetime.datetime.now()

        if not self.semver:
            _notices.append({
                "level": "WARNING",
                "message": "Ambassador has invalid version '%s'??!" % self.version
            })

        if result:
            result['cached'] = result_was_cached
        else:
            result = { 'cached': False }

        result['timestamp'] = result_timestamp.timestamp()

        # Do version & notices stuff.
        if 'latest_version' in result:
            latest_version = result['latest_version']
            latest_semver = self.get_semver(latest_version)

            if latest_semver:
                self._latest_version = latest_version
                self._latest_semver = latest_semver
            else:
                _notices.append({
                    "level": "WARNING",
                    "message": "Scout returned invalid version '%s'??!" % latest_version
                })

        if (self._latest_semver and ((not self.semver) or
                                     (self._latest_semver > self.semver))):
            _notices.append({
                "level": "INFO",
                "message": "Upgrade available! to Ambassador version %s" % self._latest_semver
            })

        if 'notices' in result:
            rnotices = typecast(List[Union[str, ScoutNotice]], result['notices'])

            for notice in rnotices:
                if isinstance(notice, str):
                    _notices.append({ "level": "WARNING", "message": notice })
                elif isinstance(notice, dict):
                    lvl = notice.get('level', 'WARNING').upper()
                    msg = notice.get('message', None)

                    if msg:
                        _notices.append({ "level": lvl, "message": msg })
                else:
                    _notices.append({ "level": "WARNING", "message": json.dumps(notice) })

        self._notices = _notices

        if self._notices:
            result['notices'] = self._notices
        else:
            result.pop('notices', None)

        return result

    @staticmethod
    def get_semver(version_string: str) -> Optional[semantic_version.Version]:
        semver = None

        try:
            semver = semantic_version.Version(version_string)
        except ValueError:
            pass

        return semver

    @staticmethod
    def parse_git_description(version: str, build: BuildInfo) -> str:
        # Here's what we get for various kinds of builds:
        #
        # Random (clean):
        # Version:               shared-dev-tgr606-f60229d
        # build.git.branch:      shared/dev/tgr606
        # build.git.commit:      f60229d
        # build.git.dirty:       False
        # build.git.description: 0.50.0-tt2-1-gf60229d
        # --> 0.50.0-local+gf60229d
        #
        # Random (dirty):
        # Version:               shared-dev-tgr606-f60229d-dirty
        # build.git.branch:      shared/dev/tgr606
        # build.git.commit:      f60229d
        # build.git.dirty:       True
        # build.git.description: 0.50.0-tt2-1-gf60229d
        # --> 0.50.0-local+gf60229d.dirty
        #
        # EA:
        # Version:               0.50.0
        # build.git.branch:      0.50.0-ea.2
        # build.git.commit:      05aefd5
        # build.git.dirty:       False
        # build.git.description: 0.50.0-ea.2
        # --> 0.50.0-ea.2
        #
        # RC
        # Version:               0.40.0
        # build.git.branch:      0.40.0-rc.1
        # build.git.commit:      d450dca
        # build.git.dirty:       False
        # build.git.description: 0.40.0-rc.1
        # --> 0.40.0-rc.1
        #
        # GA
        # Version:               0.40.0
        # build.git.branch:      0.40.0
        # build.git.commit:      e301a90
        # build.git.dirty:       False
        # build.git.description: 0.40.0
        # --> 0.40.0

        # Start by assuming that the version is sane and we needn't
        # tack any build metadata onto it.

        build_elements: List[str] = []

        m = AmbScout.reTaggedBranch.search(build.git.branch)

        if not m:
            # This isn't a proper sane version. It must be a local build. Per
            # Scout's rules, anything with semver build metadata is treated as a
            # dev version, so we need to make sure our returned version has some.
            #
            # Start by assuming that we won't find a useable version in the
            # description, and must fall back to 0.0.0.
            build_version = "0.0.0"
            desc_delta = None

            # OK. Can we find a version from what the git description starts
            # with? If so, the upgrade logic in the diagnostics will work more
            # sanely.

            m = AmbScout.reGitDescription.search(build.git.description)

            if m:
                # OK, the description ends with -$delta-g$commit at least, so
                # it may start with a version. Strip off the matching text and
                # remember the delta and commit.

                desc_delta = m.group(1)
                desc = build.git.description[0:m.start()]

                # Does the remaining description match a sane version?
                m = AmbScout.reTaggedBranch.search(desc)

                if m:
                    # Yes. Use it as the base version.
                    build_version = m.group(1)

            # We'll use prerelease "local", and include the branch and such
            # in the build metadata.
            version = '%s-local' % build_version

            # Does the commit not appear in a build element?
            build_elements = []

            if desc_delta:
                build_elements.append(desc_delta)

            build_elements.append("g%s" % build.git.commit)

            # If this branch is dirty, append a build element of "dirty".
            if build.git.dirty:
                build_elements.append('dirty')
        else:
            # The build branch is a sane version. Does it match our base version?
            (base_version, prerelease) = m.groups()

            if base_version != version:
                build_elements.append("q%s" % version.replace('.', '-'))

            # Overwrite the version with the branch only if it's important to show the
            # branch (e.g. 0.50.0-rc7 instead of 0.50.0) at all times. We should probably
            # not do this again -- or we should revamp this stuff to rebuild the Docker
            # image for a GA build.
            # version = build.git.branch

        # Finally, put it all together.
        if build_elements:
            version += "+%s" % ('.'.join(build_elements))

        return version
