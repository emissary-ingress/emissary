import errno
import json
import logging
import os
import platform
import sys
import traceback
from uuid import uuid4

import requests


class Scout:
    def __init__(
        self,
        app,
        version,
        install_id=None,
        id_plugin=None,
        id_plugin_args={},
        scout_host="metriton.datawire.io",
        **kwargs,
    ):
        """
        Create a new Scout instance for later reports.

        :param app: The application name. Required.
        :param version: The application version. Required.
        :param install_id: Optional install_id. If set, Scout will believe it.
        :param id_plugin: Optional plugin function for obtaining an install_id. See below.
        :param id_plugin_args: Optional arguments to id_plugin. See below.
        :param kwargs: Any other keyword arguments will be merged into Scout's metadata.

        If an id_plugin is present, it is called with the following parameters:

        - this Scout instance
        - the passed-in app name
        - the passed-in id_plugin_args _as keyword arguments_

        id_plugin(scout, app, **id_plugin_args)

        It must return

        - None to fall back to the default filesystem ID, or
        - a dict containing the ID and optional metadata:
           - The dict **must** have an `install_id` key with a non-empty value.
           - The dict **may** have other keys present, which will all be merged into
             Scout's `metadata`.

        If the plugin returns something invalid, Scout falls back to the default filesystem
        ID.

        See also Scout.configmap_install_id_plugin, which is an id_plugin that knows how
        to use a Kubernetes configmap (scout.config.$app) to store the install ID.

        Scout logs to the datawire.scout logger. It assumes that the logging system is
        configured to a sane default level, but you can change Scout's debug level with e.g.

        logging.getLogger("datawire.scout").setLevel(logging.DEBUG)

        """

        self.app = Scout.__not_blank("app", app)
        self.version = Scout.__not_blank("version", version)
        self.metadata = kwargs if kwargs is not None else {}
        self.user_agent = self.create_user_agent()

        self.logger = logging.getLogger("datawire.scout")

        self.install_id = install_id

        if not self.install_id and id_plugin:
            plugin_response = id_plugin(self, app, **id_plugin_args)

            self.logger.debug("Scout: id_plugin returns {0}".format(json.dumps(plugin_response)))

            if plugin_response:
                if "install_id" in plugin_response:
                    self.install_id = plugin_response["install_id"]
                    del plugin_response["install_id"]

                if plugin_response:
                    self.metadata = Scout.__merge_dicts(self.metadata, plugin_response)

        if not self.install_id:
            self.install_id = self.__filesystem_install_id(app)

        self.logger.debug("Scout using install_id {0}".format(self.install_id))

        # scout options; controlled via env vars
        self.scout_host = os.getenv("SCOUT_HOST", scout_host)
        self.use_https = os.getenv("SCOUT_HTTPS", "1").lower() in {"1", "true", "yes"}
        self.disabled = Scout.__is_disabled()

    def report(self, **kwargs):
        result = {"latest_version": self.version}

        if self.disabled:
            return result

        merged_metadata = Scout.__merge_dicts(self.metadata, kwargs)

        headers = {"User-Agent": self.user_agent}

        payload = {
            "application": self.app,
            "version": self.version,
            "install_id": self.install_id,
            "user_agent": self.create_user_agent(),
            "metadata": merged_metadata,
        }

        self.logger.debug("Scout: report payload: %s" % json.dumps(payload, indent=4))

        url = ("https://" if self.use_https else "http://") + "{}/scout".format(
            self.scout_host
        ).lower()

        try:
            resp = requests.post(url, json=payload, headers=headers, timeout=1)

            self.logger.debug("Scout: report returns %d (%s)" % (resp.status_code, resp.text))

            if resp.status_code / 100 == 2:
                result = Scout.__merge_dicts(result, resp.json())
        except OSError as e:
            self.logger.warning("Scout: could not post report: %s" % e)
            result["exception"] = "could not post report: %s" % e
        except Exception as e:
            # If scout is down or we are getting errors just proceed as if nothing happened. It should not impact the
            # user at all.
            tb = "\n".join(traceback.format_exception(*sys.exc_info()))

            result["exception"] = e
            result["traceback"] = tb

        if "new_install" in self.metadata:
            del self.metadata["new_install"]

        return result

    def create_user_agent(self):
        result = "{0}/{1} ({2}; {3}; python {4})".format(
            self.app, self.version, platform.system(), platform.release(), platform.python_version()
        ).lower()

        return result

    def __filesystem_install_id(self, app):
        config_root = os.path.join(os.path.expanduser("~"), ".config", app)
        try:
            os.makedirs(config_root)
        except OSError as ex:
            if ex.errno == errno.EEXIST and os.path.isdir(config_root):
                pass
            else:
                raise

        id_file = os.path.join(config_root, "id")
        if not os.path.isfile(id_file):
            with open(id_file, "w") as f:
                install_id = str(uuid4())
                self.metadata["new_install"] = True
                f.write(install_id)
        else:
            with open(id_file, "r") as f:
                install_id = f.read()

        return install_id

    @staticmethod
    def __not_blank(name, value):
        if value is None or str(value).strip() == "":
            raise ValueError("Value for '{}' is blank, empty or None".format(name))

        return value

    @staticmethod
    def __merge_dicts(x, y):
        z = x.copy()
        z.update(y)
        return z

    @staticmethod
    def __is_disabled():
        if str(os.getenv("TRAVIS_REPO_SLUG")).startswith("datawire/"):
            return True

        return os.getenv("SCOUT_DISABLE", "0").lower() in {"1", "true", "yes"}

    @staticmethod
    def configmap_install_id_plugin(scout, app, map_name=None, namespace="default"):
        """
        Scout id_plugin that uses a Kubernetes configmap to store the install ID.

        :param scout: Scout instance that's calling the plugin
        :param app: Name of the application that's using Scout
        :param map_name: Optional ConfigMap name to use; defaults to "scout.config.$app"
        :param namespace: Optional Kubernetes namespace to use; defaults to "default"

        This plugin assumes that the KUBERNETES_SERVICE_{HOST,PORT,PORT_HTTPS}
        environment variables are set correctly, and it assumes the default Kubernetes
        namespace unless the 'namespace' keyword argument is used to select a different
        namespace.

        If KUBERNETES_ACCESS_TOKEN is set in the environment, use that for the apiserver
        access token -- otherwise, the plugin assumes that it's running in a Kubernetes
        pod and tries to read its token from /var/run/secrets.
        """

        plugin_response = None

        if not map_name:
            map_name = "scout.config.{0}".format(app)

        kube_host = os.environ.get("KUBERNETES_SERVICE_HOST", None)

        try:
            kube_port = int(os.environ.get("KUBERNETES_SERVICE_PORT", 443))
        except ValueError:
            scout.logger.debug("Scout: KUBERNETES_SERVICE_PORT isn't numeric, defaulting to 443")
            kube_port = 443

        kube_proto = "https" if (kube_port == 443) else "http"

        kube_token = os.environ.get("KUBERNETES_ACCESS_TOKEN", None)

        if not kube_host:
            # We're not running in Kubernetes. Fall back to the usual filesystem stuff.
            scout.logger.debug("Scout: no KUBERNETES_SERVICE_HOST, not running in Kubernetes")
            return None

        if not kube_token:
            try:
                kube_token = open("/var/run/secrets/kubernetes.io/serviceaccount/token", "r").read()
            except OSError:
                pass

        if not kube_token:
            # We're not running in Kubernetes. Fall back to the usual filesystem stuff.
            scout.logger.debug("Scout: not running in Kubernetes")
            return None

        # OK, we're in a cluster. Load our map.

        base_url = "%s://%s:%s" % (kube_proto, kube_host, kube_port)
        url_path = "api/v1/namespaces/%s/configmaps" % namespace
        auth_headers = {"Authorization": "Bearer " + kube_token}
        install_id = None

        cm_url = "%s/%s" % (base_url, url_path)
        fetch_url = "%s/%s" % (cm_url, map_name)

        scout.logger.debug("Scout: trying %s" % fetch_url)

        try:
            r = requests.get(fetch_url, headers=auth_headers, verify=False)

            if r.status_code == 200:
                # OK, the map is present. What do we see?
                map_data = r.json()

                if "data" not in map_data:
                    # This is "impossible".
                    scout.logger.error("Scout: no map data in returned map???")
                else:
                    map_data = map_data.get("data", {})
                    scout.logger.debug("Scout: configmap has map data %s" % json.dumps(map_data))

                    install_id = map_data.get("install_id", None)

                    if install_id:
                        scout.logger.debug("Scout: got install_id %s from map" % install_id)
                        plugin_response = {"install_id": install_id}
        except OSError as e:
            scout.logger.debug(
                "Scout: could not read configmap (map %s, namespace %s): %s"
                % (map_name, namespace, e)
            )

        if not install_id:
            # No extant install_id. Try to create a new one.
            install_id = str(uuid4())

            cm = {
                "apiVersion": "v1",
                "kind": "ConfigMap",
                "metadata": {
                    "name": map_name,
                    "namespace": namespace,
                },
                "data": {"install_id": install_id},
            }

            scout.logger.debug("Scout: saving new install_id %s" % install_id)

            try:
                r = requests.post(cm_url, headers=auth_headers, verify=False, json=cm)

                if r.status_code == 201:
                    scout.logger.debug("Scout: saved install_id %s" % install_id)

                    plugin_response = {"install_id": install_id, "new_install": True}
                else:
                    scout.logger.error(
                        "Scout: could not save install_id: {0}, {1}".format(r.status_code, r.text)
                    )
            except OSError as e:
                logging.debug(
                    "Scout: could not write configmap (map %s, namespace %s): %s"
                    % (map_name, namespace, e)
                )

        scout.logger.debug("Scout: plugin_response %s" % json.dumps(plugin_response))
        return plugin_response
