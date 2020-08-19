from typing import Optional, Tuple

import cProfile
import difflib
import logging
import pstats

from ambassador import Cache, Config, IR
from ambassador.ir.ir import IRFileChecker
from ambassador.fetch import ResourceFetcher
from ambassador.utils import SecretHandler, NullSecretHandler, Timer

# Types
OptionalStats = Optional[pstats.Stats]

class Profiler:
    def __init__(self):
        self.pr = cProfile.Profile()
    
    def __enter__(self) -> None:
        self.pr.enable()

    def __exit__(self, *args) -> None:
        self.pr.disable()

    def stats(self) -> OptionalStats:
        return pstats.Stats(self.pr).sort_stats("tottime")        


class NullProfiler(Profiler):
    def __init__(self):
        pass
    
    def __enter__(self) -> None:
        pass

    def __exit__(self, *args) -> None:
        pass

    def stats(self) -> OptionalStats:
        return None
        

class Madness:
    def __init__(self, snapshot_path: str, 
                 logger: Optional[logging.Logger]=None,
                 secret_handler: Optional[SecretHandler]=None, 
                 file_checker: Optional[IRFileChecker]=None) -> None:
        if not logger:
            logging.basicConfig(
                level=logging.INFO,
                format="%(asctime)s madness %(levelname)s: %(message)s",
                datefmt="%Y-%m-%d %H:%M:%S"
            )

            logger = logging.getLogger('mockery')
        
        self.logger = logger

        if not secret_handler:
            secret_handler = NullSecretHandler(logger, None, None, "0")

        if not file_checker:
            file_checker = lambda f: True

        self.secret_handler = secret_handler
        self.file_checker = file_checker

        self.reset_cache()
        
        self.aconf_timer = Timer("aconf")
        self.fetcher_timer = Timer("fetcher")
        self.ir_timer = Timer("ir")

        self.aconf = Config()

        with self.fetcher_timer:
            self.fetcher = ResourceFetcher(self.logger, self.aconf)
            self.fetcher.parse_watt(open(snapshot_path, "r").read())

        with self.aconf_timer:
            self.aconf.load_all(self.fetcher.sorted())

    def reset_cache(self) -> None:
        self.cache = Cache(self.logger)

    def summarize(self) -> None:
        for timer in [
            self.fetcher_timer,
            self.aconf_timer,
            self.ir_timer,
        ]:
            if timer:
                self.logger.info(timer.summary())

    def build_ir(self,
                 cache=True, profile=False, summarize=True) -> Tuple[IR, OptionalStats]:
        self.ir_timer.reset()

        _cache = self.cache if cache else None
        _pr = Profiler() if profile else NullProfiler()
            
        with self.ir_timer:
            with _pr:
                ir = IR(self.aconf, cache=_cache,
                        secret_handler=self.secret_handler)

        if summarize:
            self.summarize()

        return (ir, _pr.stats())

    def diff(self, *rsrcs) -> None:
        jsons = [ rsrc.as_json() for rsrc in rsrcs ]

        if len(set(jsons)) == 1:
            return

        for i in range(len(rsrcs) - 1):
            if jsons[i] != jsons[i+1]:
                l1 = jsons[i].split("\n")
                l2 = jsons[i+1].split("\n")

                n1 = f"rsrcs[{i}]"
                n2 = f"rsrcs[{i+1}]"

                print("\n--------")

                for line in difflib.context_diff(l1, l2, fromfile=n1, tofile=n2):
                    line = line.rstrip()
                    print(line)
