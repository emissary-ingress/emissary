from typing import Generator, Tuple, Union

from abstract_tests import AmbassadorTest, HTTP, Node

from kat.harness import Query


class ErrorResponseOnStatusCode(AmbassadorTest):
    """
    Check that we can return a customized error response where the body is built as a formatted string.
    """

    def init(self):
        self.target = HTTP()

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        yield self, f"""
---
apiVersion: getambassador.io/v3alpha1
kind: Module
name: ambassador
ambassador_id: ["{self.ambassador_id}"]
config:
  error_response_overrides:
  - on_status_code: 401
    body:
      text_format: 'you get a 401'
  - on_status_code: 403
    body:
      text_format: 'and you get a 403'
  - on_status_code: 404
    body:
      text_format: 'cannot find the thing'
  - on_status_code: 418
    body:
      text_format: '2teapot2reply'
  - on_status_code: 500
    body:
      text_format: 'a five hundred happened'
  - on_status_code: 501
    body:
      text_format: 'very not implemented'
  - on_status_code: 503
    body:
      text_format: 'the upstream probably died'
  - on_status_code: 504
    body:
      text_format: 'took too long, sorry'
      content_type: 'apology'
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.target.path.k8s}
ambassador_id: ["{self.ambassador_id}"]
hostname: "*"
prefix: /target/
service: {self.target.path.fqdn}
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.target.path.k8s}-invalidservice
ambassador_id: ["{self.ambassador_id}"]
hostname: "*"
prefix: /target/invalidservice
service: {self.target.path.fqdn}-invalidservice
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.target.path.k8s}-invalidservice-empty
ambassador_id: ["{self.ambassador_id}"]
hostname: "*"
prefix: /target/invalidservice/empty
service: {self.target.path.fqdn}-invalidservice-empty
error_response_overrides:
- on_status_code: 503
  body:
    text_format: ''
"""

    def queries(self):
        # [0]
        yield Query(self.url("does-not-exist/"), expected=404)
        # [1]
        yield Query(
            self.url("target/"), headers={"kat-req-http-requested-status": "401"}, expected=401
        )
        # [2]
        yield Query(
            self.url("target/"), headers={"kat-req-http-requested-status": "403"}, expected=403
        )
        # [3]
        yield Query(
            self.url("target/"), headers={"kat-req-http-requested-status": "404"}, expected=404
        )
        # [4]
        yield Query(
            self.url("target/"), headers={"kat-req-http-requested-status": "418"}, expected=418
        )
        # [5]
        yield Query(
            self.url("target/"), headers={"kat-req-http-requested-status": "500"}, expected=500
        )
        # [6]
        yield Query(
            self.url("target/"), headers={"kat-req-http-requested-status": "501"}, expected=501
        )
        # [7]
        yield Query(
            self.url("target/"), headers={"kat-req-http-requested-status": "503"}, expected=503
        )
        # [8]
        yield Query(
            self.url("target/"), headers={"kat-req-http-requested-status": "504"}, expected=504
        )
        # [9]
        yield Query(self.url("target/"))
        # [10]
        yield Query(self.url("target/invalidservice"), expected=503)
        # [11]
        yield Query(self.url("target/invalidservice/empty"), expected=503)

    def check(self):
        # [0]
        assert (
            self.results[0].text == "cannot find the thing"
        ), f"unexpected response body: {self.results[0].text}"

        # [1]
        assert (
            self.results[1].text == "you get a 401"
        ), f"unexpected response body: {self.results[1].text}"

        # [2]
        assert (
            self.results[2].text == "and you get a 403"
        ), f"unexpected response body: {self.results[2].text}"

        # [3]
        assert (
            self.results[3].text == "cannot find the thing"
        ), f"unexpected response body: {self.results[3].text}"

        # [4]
        assert (
            self.results[4].text == "2teapot2reply"
        ), f"unexpected response body: {self.results[4].text}"

        # [5]
        assert (
            self.results[5].text == "a five hundred happened"
        ), f"unexpected response body: {self.results[5].text}"

        # [6]
        assert (
            self.results[6].text == "very not implemented"
        ), f"unexpected response body: {self.results[6].text}"

        # [7]
        assert (
            self.results[7].text == "the upstream probably died"
        ), f"unexpected response body: {self.results[7].text}"

        # [8]
        assert (
            self.results[8].text == "took too long, sorry"
        ), f"unexpected response body: {self.results[8].text}"
        assert self.results[8].headers["Content-Type"] == [
            "apology"
        ], f"unexpected Content-Type: {self.results[8].headers}"

        # [9] should just succeed
        assert self.results[9].text == None, f"unexpected response body: {self.results[9].text}"

        # [10] envoy-generated 503, since the upstream is 'invalidservice'.
        assert (
            self.results[10].text == "the upstream probably died"
        ), f"unexpected response body: {self.results[10].text}"

        # [11] envoy-generated 503, with an empty body override
        assert self.results[11].text == "", f"unexpected response body: {self.results[11].text}"


class ErrorResponseOnStatusCodeMappingCRD(AmbassadorTest):
    """
    Check that we can return a customized error response where the body is built as a formatted string.
    """

    def init(self):
        self.target = HTTP()

    def manifests(self) -> str:
        return (
            super().manifests()
            + f"""
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name:  {self.target.path.k8s}-crd
spec:
  ambassador_id: ["{self.ambassador_id}"]
  hostname: "*"
  prefix: /target/
  service: {self.target.path.fqdn}
  error_response_overrides:
  - on_status_code: 401
    body:
      text_format: 'you get a 401'
  - on_status_code: 403
    body:
      text_format: 'and you get a 403'
  - on_status_code: 404
    body:
      text_format: 'cannot find the thing'
  - on_status_code: 418
    body:
      text_format: '2teapot2reply'
  - on_status_code: 500
    body:
      text_format: 'a five hundred happened'
  - on_status_code: 501
    body:
      text_format: 'very not implemented'
  - on_status_code: 503
    body:
      text_format: 'the upstream probably died'
  - on_status_code: 504
    body:
      text_format: 'took too long, sorry'
      content_type: 'apology'
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: {self.target.path.k8s}-invalidservice-crd
spec:
  ambassador_id: ["{self.ambassador_id}"]
  hostname: "*"
  prefix: /target/invalidservice
  service: {self.target.path.fqdn}-invalidservice
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: {self.target.path.k8s}-invalidservice-override-crd
spec:
  ambassador_id: ["{self.ambassador_id}"]
  hostname: "*"
  prefix: /target/invalidservice/override
  service: {self.target.path.fqdn}-invalidservice
  error_response_overrides:
  - on_status_code: 503
    body:
      text_format_source:
        filename: /etc/issue
"""
        )

    def queries(self):
        # [0]
        yield Query(self.url("does-not-exist/"), expected=404)
        # [1]
        yield Query(
            self.url("target/"), headers={"kat-req-http-requested-status": "401"}, expected=401
        )
        # [2]
        yield Query(
            self.url("target/"), headers={"kat-req-http-requested-status": "403"}, expected=403
        )
        # [3]
        yield Query(
            self.url("target/"), headers={"kat-req-http-requested-status": "404"}, expected=404
        )
        # [4]
        yield Query(
            self.url("target/"), headers={"kat-req-http-requested-status": "418"}, expected=418
        )
        # [5]
        yield Query(
            self.url("target/"), headers={"kat-req-http-requested-status": "500"}, expected=500
        )
        # [6]
        yield Query(
            self.url("target/"), headers={"kat-req-http-requested-status": "501"}, expected=501
        )
        # [7]
        yield Query(
            self.url("target/"), headers={"kat-req-http-requested-status": "503"}, expected=503
        )
        # [8]
        yield Query(
            self.url("target/"), headers={"kat-req-http-requested-status": "504"}, expected=504
        )
        # [9]
        yield Query(self.url("target/"))
        # [10]
        yield Query(self.url("target/invalidservice"), expected=503)
        # [11]
        yield Query(self.url("target/invalidservice/override"), expected=503)

    def check(self):
        # [0] does not match the error response mapping, so no 404 response.
        # when envoy directly replies with 404, we see it as an empty string.
        assert self.results[0].text == "", f"unexpected response body: {self.results[0].text}"

        # [1]
        assert (
            self.results[1].text == "you get a 401"
        ), f"unexpected response body: {self.results[1].text}"

        # [2]
        assert (
            self.results[2].text == "and you get a 403"
        ), f"unexpected response body: {self.results[2].text}"

        # [3]
        assert (
            self.results[3].text == "cannot find the thing"
        ), f"unexpected response body: {self.results[3].text}"

        # [4]
        assert (
            self.results[4].text == "2teapot2reply"
        ), f"unexpected response body: {self.results[4].text}"

        # [5]
        assert (
            self.results[5].text == "a five hundred happened"
        ), f"unexpected response body: {self.results[5].text}"

        # [6]
        assert (
            self.results[6].text == "very not implemented"
        ), f"unexpected response body: {self.results[6].text}"

        # [7]
        assert (
            self.results[7].text == "the upstream probably died"
        ), f"unexpected response body: {self.results[7].text}"

        # [8]
        assert (
            self.results[8].text == "took too long, sorry"
        ), f"unexpected response body: {self.results[8].text}"
        assert self.results[8].headers["Content-Type"] == [
            "apology"
        ], f"unexpected Content-Type: {self.results[8].headers}"

        # [9] should just succeed
        assert self.results[9].text == None, f"unexpected response body: {self.results[9].text}"

        # [10] envoy-generated 503, since the upstream is 'invalidservice'.
        # this response body comes unmodified from envoy, since it goes through
        # a mapping with no error response overrides and there's no overrides
        # on the Ambassador module
        assert (
            self.results[10].text == "no healthy upstream"
        ), f"unexpected response body: {self.results[10].text}"

        # [11] envoy-generated 503, since the upstream is 'invalidservice'.
        # this response body should be matched by the `text_format_source` override
        # sorry for using /etc/issue, by the way.
        assert (
            "Welcome to Alpine Linux" in self.results[11].text
        ), f"unexpected response body: {self.results[11].text}"


class ErrorResponseReturnBodyFormattedText(AmbassadorTest):
    """
    Check that we can return a customized error response where the body is built as a formatted string.
    """

    def init(self):
        self.target = HTTP()

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        yield self, f"""
---
apiVersion: getambassador.io/v3alpha1
kind: Module
name: ambassador
ambassador_id: ["{self.ambassador_id}"]
config:
  error_response_overrides:
  - on_status_code: 404
    body:
      text_format: 'there has been an error: %RESPONSE_CODE%'
  - on_status_code: 429
    body:
      text_format: '<html>2fast %PROTOCOL%</html>'
      content_type: 'text/html'
  - on_status_code: 504
    body:
      text_format: '<html>2slow %PROTOCOL%</html>'
      content_type: 'text/html; charset="utf-8"'
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.target.path.k8s}
ambassador_id: ["{self.ambassador_id}"]
hostname: "*"
prefix: /target/
service: {self.target.path.fqdn}
"""

    def queries(self):
        # [0]
        yield Query(self.url("does-not-exist/"), expected=404)

        # [1]
        yield Query(
            self.url("target/"), headers={"kat-req-http-requested-status": "429"}, expected=429
        )

        # [2]
        yield Query(
            self.url("target/"), headers={"kat-req-http-requested-status": "504"}, expected=504
        )

    def check(self):
        # [0]
        assert (
            self.results[0].text == "there has been an error: 404"
        ), f"unexpected response body: {self.results[0].text}"
        assert self.results[0].headers["Content-Type"] == [
            "text/plain"
        ], f"unexpected Content-Type: {self.results[0].headers}"

        # [1]
        assert (
            self.results[1].text == "<html>2fast HTTP/1.1</html>"
        ), f"unexpected response body: {self.results[1].text}"
        assert self.results[1].headers["Content-Type"] == [
            "text/html"
        ], f"unexpected Content-type: {self.results[1].headers}"

        # [2]
        assert (
            self.results[2].text == "<html>2slow HTTP/1.1</html>"
        ), f"unexpected response body: {self.results[2].text}"
        assert self.results[2].headers["Content-Type"] == [
            'text/html; charset="utf-8"'
        ], f"unexpected Content-Type: {self.results[2].headers}"


class ErrorResponseReturnBodyFormattedJson(AmbassadorTest):
    """
    Check that we can return a customized error response where the body is built from a text source.
    """

    def init(self):
        self.target = HTTP()

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        yield self, f"""
---
apiVersion: getambassador.io/v3alpha1
kind: Module
name: ambassador
ambassador_id: ["{self.ambassador_id}"]
config:
  error_response_overrides:
  - on_status_code: 401
    body:
      json_format:
        error: 'unauthorized'
  - on_status_code: 404
    body:
      json_format:
        custom_error: 'truth'
        code: '%RESPONSE_CODE%'
  - on_status_code: 429
    body:
      json_format:
        custom_error: 'yep'
        toofast: 'definitely'
        code: 'code was %RESPONSE_CODE%'
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.target.path.k8s}
ambassador_id: ["{self.ambassador_id}"]
hostname: "*"
prefix: /target/
service: {self.target.path.fqdn}
"""

    def queries(self):
        yield Query(self.url("does-not-exist/"), expected=404)
        yield Query(
            self.url("target/"), headers={"kat-req-http-requested-status": "429"}, expected=429
        )
        yield Query(
            self.url("target/"), headers={"kat-req-http-requested-status": "401"}, expected=401
        )

    def check(self):
        # [0]
        # Strange gotcha: it looks like we always get an integer code here
        # even though the field specifier above is wrapped in single quotes.
        assert self.results[0].json == {
            "custom_error": "truth",
            "code": 404,
        }, f"unexpected response body: {self.results[0].json}"
        assert self.results[0].headers["Content-Type"] == [
            "application/json"
        ], f"unexpected Content-Type: {self.results[0].headers}"

        # [1]
        assert self.results[1].json == {
            "custom_error": "yep",
            "toofast": "definitely",
            "code": "code was 429",
        }, f"unexpected response body: {self.results[1].json}"
        assert self.results[1].headers["Content-Type"] == [
            "application/json"
        ], f"unexpected Content-Type: {self.results[1].headers}"

        # [2]
        assert self.results[2].json == {
            "error": "unauthorized"
        }, f"unexpected response body: {self.results[2].json}"
        assert self.results[2].headers["Content-Type"] == [
            "application/json"
        ], f"unexpected Content-Type: {self.results[2].headers}"


class ErrorResponseReturnBodyTextSource(AmbassadorTest):
    """
    Check that we can return a customized error response where the body is built as a formatted string.
    """

    def init(self):
        self.target = HTTP()

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        yield self, f"""
---
apiVersion: getambassador.io/v3alpha1
kind: Module
name: ambassador
ambassador_id: ["{self.ambassador_id}"]
config:
  error_response_overrides:
  - on_status_code: 500
    body:
      text_format_source:
        filename: '/etc/issue'
      content_type: 'application/etcissue'
  - on_status_code: 503
    body:
      text_format_source:
        filename: '/etc/motd'
      content_type: 'application/motd'
  - on_status_code: 504
    body:
      text_format_source:
        filename: '/etc/shells'
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.target.path.k8s}
ambassador_id: ["{self.ambassador_id}"]
hostname: "*"
prefix: /target/
service: {self.target.path.fqdn}
"""

    def queries(self):
        # [0]
        yield Query(
            self.url("target/"), headers={"kat-req-http-requested-status": "500"}, expected=500
        )

        # [1]
        yield Query(
            self.url("target/"), headers={"kat-req-http-requested-status": "503"}, expected=503
        )

        # [2]
        yield Query(
            self.url("target/"), headers={"kat-req-http-requested-status": "504"}, expected=504
        )

    def check(self):
        # [0] Sorry for using /etc/issue...
        print("headers = %s" % self.results[0].headers)
        assert (
            "Welcome to Alpine Linux" in self.results[0].text
        ), f"unexpected response body: {self.results[0].text}"
        assert self.results[0].headers["Content-Type"] == [
            "application/etcissue"
        ], f"unexpected Content-Type: {self.results[0].headers}"

        # [1] ...and sorry for using /etc/motd...
        assert (
            "You may change this message by editing /etc/motd." in self.results[1].text
        ), f"unexpected response body: {self.results[1].text}"
        assert self.results[1].headers["Content-Type"] == [
            "application/motd"
        ], f"unexpected Content-Type: {self.results[1].headers}"

        # [2] ...and sorry for using /etc/shells
        assert (
            "# valid login shells" in self.results[2].text
        ), f"unexpected response body: {self.results[2].text}"
        assert self.results[2].headers["Content-Type"] == [
            "text/plain"
        ], f"unexpected Content-Type: {self.results[2].headers}"


class ErrorResponseMappingBypass(AmbassadorTest):
    """
    Check that we can return a bypass custom error responses at the mapping level
    """

    def init(self):
        self.target = HTTP()

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        yield self, f"""
---
apiVersion: getambassador.io/v3alpha1
kind: Module
name: ambassador
ambassador_id: ["{self.ambassador_id}"]
config:
  error_response_overrides:
  - on_status_code: 404
    body:
      text_format: 'this is a custom 404 response'
      content_type: 'text/custom'
  - on_status_code: 418
    body:
      text_format: 'bad teapot request'
  - on_status_code: 503
    body:
      text_format: 'the upstream is not happy'
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.target.path.k8s}
ambassador_id: ["{self.ambassador_id}"]
hostname: "*"
prefix: /target/
service: {self.target.path.fqdn}
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.target.path.k8s}-invalidservice
ambassador_id: ["{self.ambassador_id}"]
hostname: "*"
prefix: /target/invalidservice
service: {self.target.path.fqdn}-invalidservice
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.target.path.k8s}-bypass
ambassador_id: ["{self.ambassador_id}"]
hostname: "*"
prefix: /bypass/
service: {self.target.path.fqdn}
bypass_error_response_overrides: true
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.target.path.k8s}-target-bypass
ambassador_id: ["{self.ambassador_id}"]
hostname: "*"
prefix: /target/bypass/
service: {self.target.path.fqdn}
bypass_error_response_overrides: true
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.target.path.k8s}-bypass-invalidservice
ambassador_id: ["{self.ambassador_id}"]
hostname: "*"
prefix: /bypass/invalidservice
service: {self.target.path.fqdn}-invalidservice
bypass_error_response_overrides: true
"""

    def queries(self):
        # [0]
        yield Query(
            self.url("bypass/"), headers={"kat-req-http-requested-status": "404"}, expected=404
        )
        # [1]
        yield Query(
            self.url("target/"), headers={"kat-req-http-requested-status": "404"}, expected=404
        )
        # [2]
        yield Query(
            self.url("target/bypass/"),
            headers={"kat-req-http-requested-status": "418"},
            expected=418,
        )
        # [3]
        yield Query(
            self.url("target/"), headers={"kat-req-http-requested-status": "418"}, expected=418
        )
        # [4]
        yield Query(self.url("target/invalidservice"), expected=503)
        # [5]
        yield Query(self.url("bypass/invalidservice"), expected=503)
        # [6]
        yield Query(
            self.url("bypass/"), headers={"kat-req-http-requested-status": "503"}, expected=503
        )
        # [7]
        yield Query(
            self.url("target/"), headers={"kat-req-http-requested-status": "503"}, expected=503
        )
        # [8]
        yield Query(self.url("bypass/"), headers={"kat-req-http-requested-status": "200"})
        # [9]
        yield Query(self.url("target/"), headers={"kat-req-http-requested-status": "200"})

    def check(self):
        # [0]
        assert self.results[0].text is None, f"unexpected response body: {self.results[0].text}"

        # [1]
        assert (
            self.results[1].text == "this is a custom 404 response"
        ), f"unexpected response body: {self.results[1].text}"
        assert self.results[1].headers["Content-Type"] == [
            "text/custom"
        ], f"unexpected Content-Type: {self.results[1].headers}"

        # [2]
        assert self.results[2].text is None, f"unexpected response body: {self.results[2].text}"

        # [3]
        assert (
            self.results[3].text == "bad teapot request"
        ), f"unexpected response body: {self.results[3].text}"

        # [4]
        assert (
            self.results[4].text == "the upstream is not happy"
        ), f"unexpected response body: {self.results[4].text}"

        # [5]
        assert (
            self.results[5].text == "no healthy upstream"
        ), f"unexpected response body: {self.results[5].text}"
        assert self.results[5].headers["Content-Type"] == [
            "text/plain"
        ], f"unexpected Content-Type: {self.results[5].headers}"

        # [6]
        assert self.results[6].text is None, f"unexpected response body: {self.results[6].text}"

        # [7]
        assert (
            self.results[7].text == "the upstream is not happy"
        ), f"unexpected response body: {self.results[7].text}"

        # [8]
        assert self.results[8].text is None, f"unexpected response body: {self.results[8].text}"

        # [9]
        assert self.results[9].text is None, f"unexpected response body: {self.results[9].text}"


class ErrorResponseMappingBypassAlternate(AmbassadorTest):
    """
    Check that we can alternate between serving a custom error response and not
    serving one. This is a baseline sanity check against Envoy's response map
    filter incorrectly persisting state across filter chain iterations.
    """

    def init(self):
        self.target = HTTP()

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        yield self, f"""
---
apiVersion: getambassador.io/v3alpha1
kind: Module
name: ambassador
ambassador_id: ["{self.ambassador_id}"]
config:
  error_response_overrides:
  - on_status_code: 404
    body:
      text_format: 'this is a custom 404 response'
      content_type: 'text/custom'
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.target.path.k8s}
ambassador_id: ["{self.ambassador_id}"]
hostname: "*"
prefix: /target/
service: {self.target.path.fqdn}
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.target.path.k8s}-invalidservice
ambassador_id: ["{self.ambassador_id}"]
hostname: "*"
prefix: /target/invalidservice
service: {self.target.path.fqdn}-invalidservice
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.target.path.k8s}-bypass
ambassador_id: ["{self.ambassador_id}"]
hostname: "*"
prefix: /bypass/
service: {self.target.path.fqdn}
bypass_error_response_overrides: true
"""

    def queries(self):
        # [0]
        yield Query(
            self.url("target/"), headers={"kat-req-http-requested-status": "404"}, expected=404
        )
        # [1]
        yield Query(
            self.url("bypass/"), headers={"kat-req-http-requested-status": "404"}, expected=404
        )
        # [2]
        yield Query(
            self.url("target/"), headers={"kat-req-http-requested-status": "404"}, expected=404
        )

    def check(self):
        # [0]
        assert (
            self.results[0].text == "this is a custom 404 response"
        ), f"unexpected response body: {self.results[0].text}"
        assert self.results[0].headers["Content-Type"] == [
            "text/custom"
        ], f"unexpected Content-Type: {self.results[0].headers}"

        # [1]
        assert self.results[1].text is None, f"unexpected response body: {self.results[1].text}"

        # [2]
        assert (
            self.results[2].text == "this is a custom 404 response"
        ), f"unexpected response body: {self.results[2].text}"
        assert self.results[2].headers["Content-Type"] == [
            "text/custom"
        ], f"unexpected Content-Type: {self.results[2].headers}"


class ErrorResponseMapping404Body(AmbassadorTest):
    """
    Check that a 404 body is consistent whether error response overrides exist or not
    """

    def init(self):
        self.target = HTTP()

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        yield self, f"""
---
apiVersion: getambassador.io/v3alpha1
kind: Module
name: ambassador
ambassador_id: ["{self.ambassador_id}"]
config:
  error_response_overrides:
  - on_status_code: 401
    body:
      text_format: 'this is a custom 401 response'
      content_type: 'text/custom'
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.target.path.k8s}
ambassador_id: ["{self.ambassador_id}"]
hostname: "*"
prefix: /target/
service: {self.target.path.fqdn}
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.target.path.k8s}-bypass
ambassador_id: ["{self.ambassador_id}"]
hostname: "*"
prefix: /bypass/
service: {self.target.path.fqdn}
bypass_error_response_overrides: true
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.target.path.k8s}-overrides
ambassador_id: ["{self.ambassador_id}"]
hostname: "*"
prefix: /overrides/
service: {self.target.path.fqdn}
error_response_overrides:
- on_status_code: 503
  body:
    text_format: 'custom 503'
"""

    def queries(self):
        # [0]
        yield Query(self.url("does-not-exist/"), expected=404)
        # [1]
        yield Query(
            self.url("target/"), headers={"kat-req-http-requested-status": "404"}, expected=404
        )
        # [2]
        yield Query(
            self.url("bypass/"), headers={"kat-req-http-requested-status": "404"}, expected=404
        )
        # [3]
        yield Query(
            self.url("overrides/"), headers={"kat-req-http-requested-status": "404"}, expected=404
        )

    def check(self):
        # [0] does not match the error response mapping, so no 404 response.
        # when envoy directly replies with 404, we see it as an empty string.
        assert self.results[0].text == "", f"unexpected response body: {self.results[0].text}"

        # [1]
        assert self.results[1].text is None, f"unexpected response body: {self.results[1].text}"

        # [2]
        assert self.results[2].text is None, f"unexpected response body: {self.results[2].text}"

        # [3]
        assert self.results[3].text is None, f"unexpected response body: {self.results[3].text}"


class ErrorResponseMappingOverride(AmbassadorTest):
    """
    Check that we can return a custom error responses at the mapping level
    """

    def init(self):
        self.target = HTTP()

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        yield self, f"""
---
apiVersion: getambassador.io/v3alpha1
kind: Module
name: ambassador
ambassador_id: ["{self.ambassador_id}"]
config:
  error_response_overrides:
  - on_status_code: 401
    body:
      text_format: 'this is a custom 401 response'
      content_type: 'text/custom'
  - on_status_code: 503
    body:
      text_format: 'the upstream is not happy'
  - on_status_code: 504
    body:
      text_format: 'the upstream took a really long time'
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.target.path.k8s}
ambassador_id: ["{self.ambassador_id}"]
hostname: "*"
prefix: /target/
service: {self.target.path.fqdn}
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.target.path.k8s}-override-401
ambassador_id: ["{self.ambassador_id}"]
hostname: "*"
prefix: /override/401/
service: {self.target.path.fqdn}
error_response_overrides:
- on_status_code: 401
  body:
    json_format:
      x: "1"
      status: '%RESPONSE_CODE%'
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.target.path.k8s}-override-503
ambassador_id: ["{self.ambassador_id}"]
hostname: "*"
prefix: /override/503/
service: {self.target.path.fqdn}
error_response_overrides:
- on_status_code: 503
  body:
    json_format:
      "y": "2"
      status: '%RESPONSE_CODE%'
"""

    def queries(self):
        # [0] Should match module's on_response_code 401
        yield Query(
            self.url("target/"), headers={"kat-req-http-requested-status": "401"}, expected=401
        )

        # [1] Should match mapping-specific on_response_code 401
        yield Query(
            self.url("override/401/"),
            headers={"kat-req-http-requested-status": "401"},
            expected=401,
        )

        # [2] Should match mapping-specific on_response_code 503
        yield Query(
            self.url("override/503/"),
            headers={"kat-req-http-requested-status": "503"},
            expected=503,
        )

        # [3] Should not match mapping-specific rule, therefore no rewrite
        yield Query(
            self.url("override/401/"),
            headers={"kat-req-http-requested-status": "503"},
            expected=503,
        )

        # [4] Should not match mapping-specific rule, therefore no rewrite
        yield Query(
            self.url("override/503/"),
            headers={"kat-req-http-requested-status": "401"},
            expected=401,
        )

        # [5] Should not match mapping-specific rule, therefore no rewrite
        yield Query(
            self.url("override/401/"),
            headers={"kat-req-http-requested-status": "504"},
            expected=504,
        )

        # [6] Should not match mapping-specific rule, therefore no rewrite
        yield Query(
            self.url("override/503/"),
            headers={"kat-req-http-requested-status": "504"},
            expected=504,
        )

        # [7] Should match module's on_response_code 503
        yield Query(
            self.url("target/"), headers={"kat-req-http-requested-status": "503"}, expected=503
        )

        # [8] Should match module's on_response_code 504
        yield Query(
            self.url("target/"), headers={"kat-req-http-requested-status": "504"}, expected=504
        )

    def check(self):
        # [0] Module's 401 rule with custom header
        assert (
            self.results[0].text == "this is a custom 401 response"
        ), f"unexpected response body: {self.results[0].text}"
        assert self.results[0].headers["Content-Type"] == [
            "text/custom"
        ], f"unexpected Content-Type: {self.results[0].headers}"

        # [1] Mapping's 401 rule with json response
        assert self.results[1].json == {
            "x": "1",
            "status": 401,
        }, f"unexpected response body: {self.results[1].json}"
        assert self.results[1].headers["Content-Type"] == [
            "application/json"
        ], f"unexpected Content-Type: {self.results[1].headers}"

        # [2] Mapping's 503 rule with json response
        assert self.results[2].json == {
            "y": "2",
            "status": 503,
        }, f"unexpected response body: {self.results[2].json}"
        assert self.results[2].headers["Content-Type"] == [
            "application/json"
        ], f"unexpected Content-Type: {self.results[2].headers}"

        # [3] Mapping has 401 rule, but response code is 503, no rewrite.
        assert self.results[3].text is None, f"unexpected response body: {self.results[3].text}"

        # [4] Mapping has 503 rule, but response code is 401, no rewrite.
        assert self.results[4].text is None, f"unexpected response body: {self.results[4].text}"

        # [5] Mapping has 401 rule, but response code is 504, no rewrite.
        assert self.results[5].text is None, f"unexpected response body: {self.results[5].text}"

        # [6] Mapping has 503 rule, but response code is 504, no rewrite.
        assert self.results[6].text is None, f"unexpected response body: {self.results[6].text}"

        # [7] Module's 503 rule, no custom header
        assert (
            self.results[7].text == "the upstream is not happy"
        ), f"unexpected response body: {self.results[7].text}"
        assert self.results[7].headers["Content-Type"] == [
            "text/plain"
        ], f"unexpected Content-Type: {self.results[7].headers}"

        # [8] Module's 504 rule, no custom header
        assert (
            self.results[8].text == "the upstream took a really long time"
        ), f"unexpected response body: {self.results[8].text}"
        assert self.results[8].headers["Content-Type"] == [
            "text/plain"
        ], f"unexpected Content-Type: {self.results[8].headers}"


class ErrorResponseSeveralMappings(AmbassadorTest):
    """
    Check that we can specify separate error response overrides on two mappings with no Module
    config
    """

    def init(self):
        self.target = HTTP()

    def manifests(self) -> str:
        return (
            super().manifests()
            + f"""
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name:  {self.target.path.k8s}-one
spec:
  ambassador_id: ["{self.ambassador_id}"]
  hostname: "*"
  prefix: /target-one/
  service: {self.target.path.fqdn}
  error_response_overrides:
  - on_status_code: 404
    body:
      text_format: '%RESPONSE_CODE% from first mapping'
  - on_status_code: 504
    body:
      text_format: 'a custom 504 response'
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: {self.target.path.k8s}-two
spec:
  ambassador_id: ["{self.ambassador_id}"]
  hostname: "*"
  prefix: /target-two/
  service: {self.target.path.fqdn}
  error_response_overrides:
  - on_status_code: 404
    body:
      text_format: '%RESPONSE_CODE% from second mapping'
  - on_status_code: 429
    body:
      text_format: 'a custom 429 response'
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: {self.target.path.k8s}-three
spec:
  ambassador_id: ["{self.ambassador_id}"]
  hostname: "*"
  prefix: /target-three/
  service: {self.target.path.fqdn}
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: {self.target.path.k8s}-four
spec:
  ambassador_id: ["{self.ambassador_id}"]
  hostname: "*"
  prefix: /target-four/
  service: {self.target.path.fqdn}
  error_response_overrides:
  - on_status_code: 500
    body:
      text_format: '500 is a bad status code'
"""
        )

    _queries = [
        {"url": "does-not-exist/", "status": 404, "text": ""},
        {"url": "target-one/", "status": 404, "text": "404 from first mapping"},
        {"url": "target-one/", "status": 429, "text": None},
        {"url": "target-one/", "status": 504, "text": "a custom 504 response"},
        {"url": "target-two/", "status": 404, "text": "404 from second mapping"},
        {"url": "target-two/", "status": 429, "text": "a custom 429 response"},
        {"url": "target-two/", "status": 504, "text": None},
        {"url": "target-three/", "status": 404, "text": None},
        {"url": "target-three/", "status": 429, "text": None},
        {"url": "target-three/", "status": 504, "text": None},
        {"url": "target-four/", "status": 404, "text": None},
        {"url": "target-four/", "status": 429, "text": None},
        {"url": "target-four/", "status": 504, "text": None},
    ]

    def queries(self):
        for x in self._queries:
            yield Query(
                self.url(x["url"]),
                headers={"kat-req-http-requested-status": str(x["status"])},
                expected=x["status"],
            )

    def check(self):
        for i in range(len(self._queries)):
            expected = self._queries[i]["text"]
            res = self.results[i]
            assert (
                res.text == expected
            ), f'unexpected response body on query {i}: "{res.text}", wanted "{expected}"'
