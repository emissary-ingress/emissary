from typing import ClassVar, Dict, List, Optional, Tuple, Union

import datetime
import json
import logging
import re
import os

import semantic_version

from scout import Scout

# Import version stuff directly from ambassador.VERSION to avoid a circular import.
from .VERSION import Version, Build, BuildInfo

ScoutNotice = Union[str, Dict[str, str]]


class AmbScout:
    reTaggedBranch: ClassVar = re.compile(r'^(\d+\.\d+\.\d+)(-[a-zA-Z][a-zA-Z]\d+)?$')
    reGitDescription: ClassVar = re.compile(r'-(\d+)-g([0-9a-f]+)$')

    install_id: str
    runtime: str
    namespace: str

    version: str
    semver: Optional[semantic_version.Version]

    _scout: Optional[Scout]
    _scout_error: Optional[str]

    _notices: Optional[List[ScoutNotice]]
    _last_result: Optional[dict]
    _last_update: Optional[datetime.datetime]
    _update_frequency: Optional[datetime.timedelta]

    _latest_version: Optional[str] = None
    _latest_semver: Optional[semantic_version.Version] = None

    def __init__(self) -> None:
        self.install_id = os.environ.get('AMBASSADOR_SCOUT_ID', "00000000-0000-0000-0000-000000000000")
        self.runtime = "kubernetes" if os.environ.get('KUBERNETES_SERVICE_HOST', None) else "docker"
        self.namespace = os.environ.get('AMBASSADOR_NAMESPACE', 'default')

        self.version = self.parse_git_description(Version, Build)
        self.semver = self.get_semver(self.version)

        self.logger = logging.getLogger("ambassador.scout")
        self.logger.setLevel(logging.DEBUG)

        self.logger.debug("Ambassador version %s" % Version)
        self.logger.debug("Scout version      %s%s" % (self.version, " - BAD SEMVER" if not self.semver else ""))
        self.logger.debug("Runtime            %s" % self.runtime)
        self.logger.debug("Namespace          %s" % self.namespace)

        self._scout = None
        self._scout_error = None

        self._notices = None
        self._last_result = None
        self._last_update = datetime.datetime.now() - datetime.timedelta(hours=24)
        self._update_frequency = datetime.timedelta(hours=4)
        self._latest_version = None
        self._latest_semver = None

    def __str__(self) -> str:
        return ("%s: %s" % ("OK" if self._scout else "??", 
                            self._scout_error if self._scout_error else "OK"))

    @property
    def scout(self) -> Optional[Scout]:
        if not self._scout:
            try:
                self._scout = Scout(app="ambassador", version=self.version, install_id=self.install_id)
                self._scout_error = None
                self.logger.debug("Scout connection established")
            except OSError as e:
                self._scout = None
                self._scout_error = e
                self.logger.debug("Scout connection failed, will retry later: %s" % self._scout_error)

        return self._scout

    def report(self, force_result: Optional[dict]=None, **kwargs) -> Tuple[dict, List[ScoutNotice]]:
        _notices: List[ScoutNotice] = []

        env_result = os.environ.get("AMBASSADOR_SCOUT_RESULT", None)
        if env_result:
            force_result = json.loads(env_result)

        result: Optional[dict] = force_result
        result_was_cached: bool = False

        if not result:
            if 'runtime' not in kwargs:
                kwargs['runtime'] = self.runtime

            if 'namespace' not in kwargs:
                kwargs['namespace'] = self.namespace

            # How long since the last Scout update? If it's been more than an hour,
            # check Scout again.

            now = datetime.datetime.now()

            if (now - self._last_update) > self._update_frequency:
                if self.scout:
                    result = self.scout.report(**kwargs)

                    self._last_update = now
                    self._last_result = dict(**result)
                else:
                    result = { "scout": "unavailable: %s" % self._scout_error }
                    _notices.append({ "level": "debug",
                                      "message": "scout temporarily unavailable: %s" % self._scout_error })

                # Whether we could talk to Scout or not, update the timestamp so we don't
                # try again too soon.
                result_timestamp = datetime.datetime.now()
            else:
                _notices.append({ "level": "debug", "message": "Returning cached result" })
                result = dict(**self._last_result)
                result_was_cached = True

                result_timestamp = self._last_update
        else:
            _notices.append({ "level": "debug", "message": "Returning forced result" })
            result_timestamp = datetime.datetime.now()

        if not self.semver:
            _notices.append({
                "level": "warning",
                "message": "Ambassador has invalid version '%s'??!" % self.version
            })

        result['cached'] = result_was_cached
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
                    "level": "warning",
                    "message": "Scout returned invalid version '%s'??!" % latest_version
                })

        if (self._latest_semver and ((not self.semver) or
                                     (self._latest_semver > self.semver))):
            _notices.append({
                "level": "info",
                "message": "Upgrade available! to Ambassador version %s" % self._latest_semver
            })

        if 'notices' in result:
            _notices.extend(result['notices'])

        self._notices = _notices

        return result, self._notices

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
        # build.git.branch:      0.50.0-ea2
        # build.git.commit:      05aefd5
        # build.git.dirty:       False
        # build.git.description: 0.50.0-ea2
        # --> 0.50.0-ea2
        #
        # RC
        # Version:               0.40.0
        # build.git.branch:      0.40.0-rc1
        # build.git.commit:      d450dca
        # build.git.dirty:       False
        # build.git.description: 0.40.0-rc1
        # --> 0.40.0-rc1
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

        build_elements = []

        if not AmbScout.reTaggedBranch.search(version):
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

        # Finally, put it all together.
        if build_elements:
            version += "+%s" % ('.'.join(build_elements))

        return version
