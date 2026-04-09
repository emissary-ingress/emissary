import logging
from types import SimpleNamespace

import ambassador_diag.diagd as diagd


logger = logging.getLogger("ambassador")


CRD_ERROR = (
    "Ambassador could not find core CRD definitions. Please visit "
    "https://www.getambassador.io/docs/edge-stack/latest/topics/install/upgrade-to-edge-stack/#5-update-and-restart "
    "for more information. You can continue using Ambassador via Kubernetes annotations, any configuration via CRDs will be ignored..."
)

TLS_NOTICE = "No TLS termination and no fallback cert -- defaulting to cleartext-only."


class FakeNotices:
    def __init__(self) -> None:
        self.notices = []

    def post(self, notice) -> None:
        self.notices.append(notice)

    def prepend(self, notice) -> None:
        self.notices.insert(0, notice)


class FakeDiag:
    def __init__(self, errors=None, notices=None) -> None:
        self._errors = errors or {}
        self._notices = notices or {}

    def as_dict(self):
        return {"errors": self._errors, "notices": self._notices}


def make_ir(*, diagnostics=None, errors=None, notices=None, tls_contexts=None, mapping_count=1):
    mappings = [{"prefix": "/qotm/", "name": "qotm"} for _ in range(mapping_count)]
    groups = {"group": SimpleNamespace(mappings=mappings)} if mapping_count else {}

    return SimpleNamespace(
        ambassador_module=SimpleNamespace(diagnostics=diagnostics or {}),
        aconf=SimpleNamespace(errors=errors or {}, notices=notices or {}),
        tls_contexts=tls_contexts or [],
        groups=groups,
    )


def test_collect_errors_and_notices_filters_suppressed_ui_messages(monkeypatch):
    fake_app = SimpleNamespace(ir=make_ir(diagnostics={"missing_tls_ok": True}), notices=FakeNotices())
    monkeypatch.setattr(diagd, "app", fake_app)

    diag = FakeDiag(
        errors={"-global-": [{"error": CRD_ERROR}, {"error": "real error"}]},
        notices={"diag": [TLS_NOTICE, "keep this notice"]},
    )

    result = diagd.collect_errors_and_notices(SimpleNamespace(args={}), "reqid", "overview", diag)

    assert result["errors"] == [("", "real error")]
    assert fake_app.notices.notices == [{"level": "NOTICE", "message": "diag: keep this notice"}]


def test_check_environment_allows_missing_tls_when_configured():
    watcher = diagd.AmbassadorEventWatcher(SimpleNamespace(logger=logger))

    watcher.check_environment(make_ir(diagnostics={"missing_tls_ok": True}))

    assert watcher.env_good is True
    assert watcher.failure_list == []
    assert watcher.env_status.to_dict()["TLS"] == {
        "status": True,
        "specifics": [
            (True, "No TLSContexts are active, but diagnostics.missing_tls_ok allows that")
        ],
    }


def test_check_environment_ignores_suppressed_crd_errors():
    watcher = diagd.AmbassadorEventWatcher(SimpleNamespace(logger=logger))
    ir = make_ir(
        diagnostics={"missing_tls_ok": True},
        errors={"-global-": [{"error": CRD_ERROR}]},
    )

    watcher.check_environment(ir)

    assert watcher.env_good is True
    assert watcher.failure_list == []
    assert watcher.env_status.to_dict()["Error check"] == {
        "status": True,
        "specifics": [(True, "No errors logged")],
    }


def test_check_environment_still_flags_real_errors():
    watcher = diagd.AmbassadorEventWatcher(SimpleNamespace(logger=logger))
    ir = make_ir(
        diagnostics={"missing_tls_ok": True},
        errors={"-global-": [{"error": "a real error"}]},
    )

    watcher.check_environment(ir)

    assert watcher.env_good is False
    assert watcher.failure_list == []
    assert watcher.env_status.to_dict()["Error check"] == {
        "status": False,
        "specifics": [(False, "1 total error logged")],
    }
