from pathlib import Path

__all__ = ("TEMPLATE_PATH",)

TEMPLATE_PATH: Path = Path(__file__).parent.resolve()
