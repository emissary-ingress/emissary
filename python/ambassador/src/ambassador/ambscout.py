import datetime
import logging
import os
import re
from typing import Any, ClassVar, Dict, List, Optional, Union
from typing import cast as typecast

import semantic_version

from .scout import Scout
from .utils import dump_json, parse_json
from .VERSION import Commit, Version

ScoutNotice = Dict[str, str]


class LocalScout:
    def __init__(self, logger, app: str, version: str, install_id: str) -> None:
        self.logger = logger
        self.app = app
        self.version = version
        self.install_id = install_id

        self.events: List[Dict[str, Any]] = []

        self.logger.info(
            f"LocalScout: initialized for {app} {version}: ID {install_id}"
        )

    def report(self, **kwargs) -> dict:
        self.events.append(kwargs)

        mode = kwargs["mode"]
        action = kwargs["action"]

        now = datetime.datetime.now().timestamp()

        kwargs["local_scout_timestamp"] = now

        if "timestamp" not in kwargs:
            kwargs["timestamp"] = now

        self.logger.info(f"LocalScout: mode {mode}, action {action} ({kwargs})")

        return {
            "latest_version": self.version,
            "application": self.app,
            "cached": False,
            "notices": [
                {"level": "WARNING", "message": "Using LocalScout, result is faked!"}
            ],
            "timestamp": now,
        }

    def reset_events(self) -> None:
        self.events = []


class AmbScout:
    reTaggedBranch: ClassVar = re.compile(
        r"^v?(\d+\.\d+\.\d+)(-[a-zA-Z][a-zA-Z]\.\d+)?$"
    )
    reGitDescription: ClassVar = re.compile(r"-(\d+)-g([0-9a-f]+)$")

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

    def __init__(
        self,
        install_id=None,
        update_frequency=datetime.timedelta(hours=12),
        local_only=False,
    ) -> None:
        if not install_id:
            install_id = os.environ.get(
                "AMBASSADOR_CLUSTER_ID",
                os.environ.get(
                    "AMBASSADOR_SCOUT_ID", "00000000-0000-0000-0000-000000000000"
                ),
            )

        self.install_id = install_id
        self.runtime = (
            "kubernetes"
            if os.environ.get("KUBERNETES_SERVICE_HOST", None)
            else "docker"
        )
        self.namespace = os.environ.get("AMBASSADOR_NAMESPACE", "default")

        self.app = "ambassador"
        self.version = Version
        self.semver = self.get_semver(self.version)

        self.logger = logging.getLogger("ambassador.scout")
        # self.logger.setLevel(logging.DEBUG)

        self.logger.debug("Ambassador version %s built from %s" % (Version, Commit))
        self.logger.debug(
            "Scout version      %s%s"
            % (self.version, " - BAD SEMVER" if not self.semver else "")
        )
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
            assert self._scout
            typecast(LocalScout, self._scout).reset_events()

    def __str__(self) -> str:
        return "%s: %s" % (
            "OK" if self._scout else "??",
            self._scout_error if self._scout_error else "OK",
        )

    @property
    def scout(self) -> Optional[Union[Scout, LocalScout]]:
        if not self._scout:
            if self._local_only:
                self._scout = LocalScout(
                    logger=self.logger,
                    app=self.app,
                    version=self.version,
                    install_id=self.install_id,
                )
                self.logger.debug("LocalScout initialized")
            else:
                try:
                    self._scout = Scout(
                        app=self.app, version=self.version, install_id=self.install_id
                    )
                    self._scout_error = None
                    self.logger.debug("Scout connection established")
                except OSError as e:
                    self._scout = None
                    self._scout_error = str(e)
                    self.logger.debug(
                        "Scout connection failed, will retry later: %s"
                        % self._scout_error
                    )

        return self._scout

    def report(
        self, force_result: Optional[dict] = None, no_cache=False, **kwargs
    ) -> dict:
        _notices: List[ScoutNotice] = []

        # Silly, right?
        use_cache = not no_cache

        env_result = None

        if use_cache:
            env_result = os.environ.get("AMBASSADOR_SCOUT_RESULT", None)

            if env_result:
                force_result = parse_json(env_result)

        result: Optional[dict] = force_result
        result_was_cached: bool = False

        if not result:
            if "runtime" not in kwargs:
                kwargs["runtime"] = self.runtime

            if "commit" not in kwargs:
                kwargs["commit"] = Commit

            # How long since the last Scout update? If it's been more than an hour,
            # check Scout again.

            now = datetime.datetime.now()

            needs_update = True

            if use_cache:
                if self._last_update:
                    since_last_update = now - typecast(
                        datetime.datetime, self._last_update
                    )
                    needs_update = since_last_update > self._update_frequency

            if needs_update:
                if self.scout:
                    result = self.scout.report(**kwargs)

                    self._last_update = now
                    self._last_result = (
                        dict(**typecast(dict, result)) if result else None
                    )
                else:
                    result = {"scout": "unavailable: %s" % self._scout_error}
                    _notices.append(
                        {
                            "level": "DEBUG",
                            "message": "scout temporarily unavailable: %s"
                            % self._scout_error,
                        }
                    )

                # Whether we could talk to Scout or not, update the timestamp so we don't
                # try again too soon.
                result_timestamp = datetime.datetime.now()
            else:
                _notices.append(
                    {"level": "DEBUG", "message": "Returning cached result"}
                )
                result = (
                    dict(**typecast(dict, self._last_result))
                    if self._last_result
                    else None
                )
                result_was_cached = True

                # We can't get here unless self._last_update is set.
                result_timestamp = typecast(datetime.datetime, self._last_update)
        else:
            _notices.append(
                {"level": "INFO", "message": "Returning forced Scout result"}
            )
            result_timestamp = datetime.datetime.now()

        if not self.semver:
            _notices.append(
                {
                    "level": "WARNING",
                    "message": "Ambassador has invalid version '%s'??!" % self.version,
                }
            )

        if result:
            result["cached"] = result_was_cached
        else:
            result = {"cached": False}

        result["timestamp"] = result_timestamp.timestamp()

        # Do version & notices stuff.
        if "latest_version" in result:
            latest_version = result["latest_version"]
            latest_semver = self.get_semver(latest_version)

            if latest_semver:
                self._latest_version = latest_version
                self._latest_semver = latest_semver
            else:
                _notices.append(
                    {
                        "level": "WARNING",
                        "message": "Scout returned invalid version '%s'??!"
                        % latest_version,
                    }
                )

        if self._latest_semver and (
            (not self.semver) or (self._latest_semver > self.semver)
        ):
            _notices.append(
                {
                    "level": "INFO",
                    "message": "Upgrade available! to Ambassador version %s"
                    % self._latest_semver,
                }
            )

        if "notices" in result:
            rnotices = typecast(List[Union[str, ScoutNotice]], result["notices"])

            for notice in rnotices:
                if isinstance(notice, str):
                    _notices.append({"level": "WARNING", "message": notice})
                elif isinstance(notice, dict):
                    lvl = notice.get("level", "WARNING").upper()
                    msg = notice.get("message", None)

                    if msg:
                        _notices.append({"level": lvl, "message": msg})
                else:
                    _notices.append({"level": "WARNING", "message": dump_json(notice)})

        self._notices = _notices

        if self._notices:
            result["notices"] = self._notices
        else:
            result.pop("notices", None)

        return result

    @staticmethod
    def get_semver(version_string: str) -> Optional[semantic_version.Version]:
        semver = None

        try:
            semver = semantic_version.Version(version_string)
        except ValueError:
            pass

        return semver
