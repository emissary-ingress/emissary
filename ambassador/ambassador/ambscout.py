import os

from scout import Scout

class AmbScout:
    def __init__(self) -> None:
        scout_install_id = os.environ.get('AMBASSADOR_SCOUT_ID', "00000000-0000-0000-0000-000000000000")

        try:
            self._scout = Scout(app="ambassador", version="0.0.1-crap", install_id=scout_install_id)
            self._scout_error = None
        except OSError as e:
            self._scout_error = e

    def __str__(self) -> str:
        return ("%s: %s" % ("OK" if self._scout else "??", 
                            self._scout_error if self._scout_error else "OK"))

# def get_semver(what, version_string):
#     semver = None

#     try:
#         semver = semantic_version.Version(version_string)
#     except ValueError:
#         pass

#     return semver

#     # Weird stuff. The build version looks like
#     #
#     # 0.12.0                    for a prod build, or
#     # 0.12.1-b2.da5d895.DIRTY   for a dev build (in this case made from a dirty true)
#     #
#     # Now:
#     # - Scout needs a build number (semver "+something") to flag a non-prod release;
#     #   but
#     # - DockerHub cannot use a build number at all; but
#     # - 0.12.1-b2 comes _before_ 0.12.1+b2 in SemVer land.
#     #
#     # FFS.
#     #
#     # We cope with this by transforming e.g.
#     #
#     # 0.12.1-b2.da5d895.DIRTY into 0.12.1-b2+da5d895.DIRTY
#     #
#     # for Scout.

#     scout_version = Version

#     if '-' in scout_version:
#         # TODO(plombardi): This version code needs to be rewritten. We should only report RC and GA versions.
#         #
#         # As of the time when we moved to streamlined branch, merge and release model the way versions in development
#         # land are rendered has changed. A development version no longer has any <MAJOR>.<MINOR>.<PATCH> information and
#         # is instead rendered as <BRANCH_NAME>-<GIT_SHORT_HASH>[-dirty] where [-dirty] is only appended for modified
#         # source trees.
#         #
#         # Long term we are planning to remove the version report for development branches anyways so all of this
#         # formatting for versions

#         scout_version = "0.0.0-" + Version.split("-")[1]  # middle part is commit hash
#         # Dev build!
#         # v, p = scout_version.split('-')
#         # p, b = p.split('.', 1) if ('.' in p) else (0, p)
#         #
#         # scout_version = "%s-%s+%s" % (v, p, b)

#     # Use scout_version here, not __version__, because the version
#     # coming back from Scout will use build numbers for dev builds, but
#     # __version__ won't, and we need to be consistent for comparison.
#     current_semver = get_semver("current", scout_version)

# Moved to config.py
##     # When using multiple Ambassadors in one cluster, use AMBASSADOR_ID to distinguish them.
##     ambassador_id = os.environ.get('AMBASSADOR_ID', 'default')

##     runtime = "kubernetes" if os.environ.get('KUBERNETES_SERVICE_HOST', None) else "docker"
##     namespace = os.environ.get('AMBASSADOR_NAMESPACE', 'default')

#     # Default to using the Nil UUID unless the environment variable is set explicitly
#     scout_install_id = os.environ.get('AMBASSADOR_SCOUT_ID', "00000000-0000-0000-0000-000000000000")

#     try:
#         scout = Scout(app="ambassador", version=scout_version, install_id=scout_install_id)
#         scout_error = None
#     except OSError as e:
#         scout_error = e

#     scout_latest_version = None
#     scout_latest_semver = None
#     scout_notices = []

#     scout_last_response = None
#     scout_last_update = datetime.datetime.now() - datetime.timedelta(hours=24)
#     scout_update_frequency = datetime.timedelta(hours=4)

#     @classmethod
#     def scout_report(klass, force_result=None, **kwargs):
#         _notices = []

#         env_result = os.environ.get("AMBASSADOR_SCOUT_RESULT", None)
#         if env_result:
#             force_result = json.loads(env_result)

#         result = force_result
#         result_timestamp = None
#         result_was_cached = False

#         if not result:
#             if Config.scout:
#                 if 'runtime' not in kwargs:
#                     kwargs['runtime'] = Config.runtime

#                 # How long since the last Scout update? If it's been more than an hour,
#                 # check Scout again.

#                 now = datetime.datetime.now()

#                 if (now - Config.scout_last_update) > Config.scout_update_frequency:
#                     result = Config.scout.report(**kwargs)

#                     Config.scout_last_update = now
#                     Config.scout_last_result = dict(**result)
#                 else:
#                     # _notices.append({ "level": "debug", "message": "Returning cached result" })
#                     result = dict(**Config.scout_last_result)
#                     result_was_cached = True

#                 result_timestamp = Config.scout_last_update
#             else:
#                 result = { "scout": "unavailable" }
#                 result_timestamp = datetime.datetime.now()
#         else:
#             _notices.append({ "level": "debug", "message": "Returning forced result" })
#             result_timestamp = datetime.datetime.now()

#         if not Config.current_semver:
#             _notices.append({
#                 "level": "warning",
#                 "message": "Ambassador has bad version '%s'??!" % Config.scout_version
#             })

#         result['cached'] = result_was_cached
#         result['timestamp'] = result_timestamp.timestamp()

#         # Do version & notices stuff.
#         if 'latest_version' in result:
#             latest_version = result['latest_version']
#             latest_semver = get_semver("latest", latest_version)

#             if latest_semver:
#                 Config.scout_latest_version = latest_version
#                 Config.scout_latest_semver = latest_semver
#             else:
#                 _notices.append({
#                     "level": "warning",
#                     "message": "Scout returned bad version '%s'??!" % latest_version
#                 })

#         if (Config.scout_latest_semver and
#             ((not Config.current_semver) or
#              (Config.scout_latest_semver > Config.current_semver))):
#             _notices.append({
#                 "level": "info",
#                 "message": "Upgrade available! to Ambassador version %s" % Config.scout_latest_semver
#             })

#         if 'notices' in result:
#             _notices.extend(result['notices'])

#         Config.scout_notices = _notices

#         return result
