#!python

# Copyright 2020 Datawire. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License

import datetime
import logging
import time
from typing import List, Optional, Tuple

PerfCounter = float


class ReconfigStats:
    """
    Track metrics for reconfigurations, whether complete or incremental.
    There's a surprising amount of business logic in here -- read carefully
    before messing with this!
    """

    def __init__(
        self,
        logger: logging.Logger,
        max_incr_between_checks=100,
        max_time_between_checks=600,
        max_config_between_timers=10,
        max_time_between_timers=120,
    ) -> None:
        """
        Initialize this ReconfigStats.

        :param max_incr_between_checks: Maximum number of outstanding incrementals before a sanity check
        :param max_time_between_checks: Maximum number of seconds between sanity checks
        :param max_config_between_timers: Maximum number of configurations before logging timers
        :param max_time_between_timers: Maximum number of seconds between logging timers
        """

        # Save config elements.
        self.logger = logger
        self.max_incr_between_checks = max_incr_between_checks
        self.max_time_between_checks = max_time_between_checks
        self.max_config_between_timers = max_config_between_timers
        self.max_time_between_timers = max_time_between_timers

        # self.reconfigures tracks the last few reconfigures, both for business
        # logic around the last reconfigure and for logging.
        self.reconfigures: List[Tuple[str, PerfCounter]] = []

        # self.counts tracks how many of each kind of reconfiguration have
        # happened, for metrics.
        self.counts = {"incremental": 0, "complete": 0}

        # In many cases, the previous complete reconfigure will have fallen out
        # of self.reconfigures, so we remember its timestamp separately.
        self.last_complete: Optional[PerfCounter] = None

        # Likewise, remember the time of the last sanity check...
        self.last_check: Optional[PerfCounter] = None

        # ...and the time of the last timer logs.
        self.last_timer_log: Optional[PerfCounter] = None

        # self.incrementals_outstanding is the number of incrementals since the
        # last complete. Once too many incrementals pile up, we do a sanity check.
        self.incrementals_outstanding = 0

        # self.configs_outstanding is the number of configurations (either kind)
        # since we last logged the timers.  Once too many configurations pile up,
        # we log the timers.
        self.configs_outstanding = 0

        # self.checks is how many sanity checks we've done. self.errors is how many
        # of them failed.
        self.checks = 0
        self.errors = 0

    def mark(self, what: str, when: Optional[PerfCounter] = None) -> None:
        """
        Mark that a reconfigure has occurred. The 'what' parameter is one of
        "complete" for a complete reconfigure, "incremental" for an incremental,
        or "diag" to indicate that we're not really reconfiguring, we just generated
        the diagnostics so may need to log timers.

        :param what: "complete", "incremental", or "diag".
        :param when: The time at which this occurred. Can be None, meaning "now".
        """

        if not when:
            when = time.perf_counter()

        if (what == "incremental") and not self.last_complete:
            # You can't have an incremental without a complete to start.
            # If this is the first reconfigure, it's a complete reconfigure.
            what = "complete"

        # Should we update all the counters?
        update_counters = True

        if what == "complete":
            # For a complete reconfigure, we need to clear all the outstanding
            # incrementals, and also remember when it happened.
            self.incrementals_outstanding = 0
            self.last_complete = when

            # A complete reconfigure also resets the last check time, because
            # we consider the complete reconfigure to be a sanity check, basically.
            # Note that it does _not_ reset any timer-logging stuff.
            self.last_check = when

            self.logger.debug(f"MARK COMPLETE @ {when}")
        elif what == "incremental":
            # For an incremental reconfigure, we need to remember that we have
            # one more incremental outstanding.
            self.incrementals_outstanding += 1

            self.logger.debug(f"MARK INCREMENTAL @ {when}")
        elif what == "diag":
            # Don't update all the counters for a diagnostic update.
            update_counters = False
        else:
            raise RuntimeError(f"ReconfigStats: unknown reconfigure type {what}")

        # If we should update the counters...
        if update_counters:
            # ...then update the counts and our reconfigures list.
            self.counts[what] += 1
            self.reconfigures.append((what, when))

            if len(self.reconfigures) > 10:
                self.reconfigures.pop(0)

        # In all cases, update the number of outstanding configurations. This will
        # trigger timer logging for diagnostics updates.
        self.configs_outstanding += 1

    def needs_check(self, when: Optional[PerfCounter] = None) -> bool:
        """
        Determine if we need to do a complete reconfigure to doublecheck our
        incrementals. The logic here is that we need a check every 100 incrementals
        or every 10 minutes, whichever comes first.

        :param when: Override the effective time of the check. Primarily useful for testing.
        :return: True if a check is needed, False if not
        """

        if not when:
            when = time.perf_counter()

        if len(self.reconfigures) == 0:
            # No reconfigures, so no need to check.
            # self.logger.debug(f"NEEDS_CHECK @ {when}: no reconfigures, skip")
            return False

        # Grab information about our last reconfiguration.
        what, _ = self.reconfigures[-1]

        if what == "complete":
            # Last reconfiguration was a complete reconfiguration, so
            # no need to check.
            # self.logger.debug(f"NEEDS_CHECK @ {when}: last was complete, skip")
            return False

        if self.incrementals_outstanding == 0:
            # If we have a bunch of incrementals, then we do a check, we can land
            # here with no outstanding incrementals, in which case it's pointless to
            # do a check.
            # self.logger.debug(f"NEEDS_CHECK @ {when}: outstanding 0, skip")
            return False

        # OK, the last one was an incremental, which implies that we must have some
        # outstanding incrementals. Have we hit our maximum between checks?
        if self.incrementals_outstanding >= self.max_incr_between_checks:
            # Yup, time to check.
            # self.logger.debug(f"NEEDS_CHECK @ {when}: outstanding {self.incrementals_outstanding}, check")
            return True

        # self.logger.debug(f"NEEDS_CHECK @ {when}: outstanding {self.incrementals_outstanding}")

        # We're good for outstanding incrementals. How about the max time between checks?
        # (We must have a last check time - which may be the time of the last complete
        # reconfigure, of course - to go on at this point.)
        assert self.last_check is not None

        delta = when - self.last_check

        if delta > self.max_time_between_checks:
            # Yup, it's been long enough.
            # self.logger.debug(f"NEEDS_CHECK @ {when}: delta {delta}, check")
            return True

        # self.logger.debug(f"NEEDS_CHECK @ {when}: delta {delta}, skip")
        return False

    def needs_timers(self, when: Optional[PerfCounter] = None) -> bool:
        """
        Determine if we need to log the timers or not. The logic here is that
        we need to log every max_configs_between_timers incrementals or every
        or every max_time_between_timers seconds, whichever comes first.

        :param when: Override the effective time of the check. Primarily useful for testing.
        :return: True if we need to log timers, False if not
        """

        if not when:
            when = time.perf_counter()

        if len(self.reconfigures) == 0:
            # No reconfigures, so no need to check.
            # self.logger.debug(f"NEEDS_TIMERS @ {when}: no reconfigures, skip")
            return False

        # If we have no configurations outstanding, we're done.
        if self.configs_outstanding == 0:
            # self.logger.debug(f"NEEDS_TIMERS @ {when}: outstanding 0, skip")
            return False

        # Have we hit our maximum number of outstanding configurations?
        if self.configs_outstanding >= self.max_config_between_timers:
            # Yup, time to log.
            # self.logger.debug(f"NEEDS_TIMERS @ {when}: outstanding {self.configs_outstanding}, check")
            return True

        # self.logger.debug(f"NEEDS_TIMERS @ {when}: outstanding {self.configs_outstanding}")

        # We're good for outstanding incrementals. How about the max time between timers?
        # Note that we may _never_ have logged timers before -- if that's the case, use
        # the time of our last complete reconfigure, which must always be set, as a
        # baseline.

        assert self.last_complete is not None

        baseline = self.last_timer_log or self.last_complete
        delta = when - baseline

        if delta > self.max_time_between_timers:
            # Yup, it's been long enough.
            # self.logger.debug(f"NEEDS_TIMERS @ {when}: delta {delta}, check")
            return True

        # self.logger.debug(f"NEEDS_TIMERS @ {when}: delta {delta}, skip")
        return False

    def mark_checked(self, result: bool, when: Optional[PerfCounter] = None) -> None:
        """
        Mark that we have done a check, and note the results. This resets our
        outstanding incrementals to 0, and also resets our last check time.

        :param result: True if the check was good, False if not
        :param when: Override the effective time. Primarily useful for testing.
        """

        self.logger.debug(f"MARK_CHECKED @ {when}: {result}")

        self.incrementals_outstanding = 0
        self.checks += 1

        if not result:
            self.errors += 1

        self.last_check = when or time.perf_counter()

    def mark_timers_logged(self, when: Optional[PerfCounter] = None) -> None:
        """
        Mark that we have logged timers. This resets our outstanding configurations
        to 0, and also resets our last timer log time.

        :param when: Override the effective time. Primarily useful for testing.
        """

        self.logger.debug(f"MARK_TIMERS @ {when}")

        self.configs_outstanding = 0
        self.last_timer_log = when or time.perf_counter()

    @staticmethod
    def isofmt(when: Optional[PerfCounter], now_pc: PerfCounter, now_dt: datetime.datetime) -> str:
        if not when:
            return "(none)"

        delta = datetime.timedelta(seconds=when - now_pc)
        when_dt = now_dt + delta

        return when_dt.isoformat()

    def dump(self) -> None:
        now_pc = time.perf_counter()
        now_dt = datetime.datetime.now()

        for what, when in self.reconfigures:
            self.logger.info(f"CACHE: {what} reconfigure at {self.isofmt(when, now_pc, now_dt)}")

        for what in ["incremental", "complete"]:
            self.logger.info(f"CACHE: {what} count: {self.counts[what]}")

        self.logger.info(f"CACHE: incrementals outstanding: {self.incrementals_outstanding}")
        self.logger.info(f"CACHE: incremental checks: {self.checks}, errors {self.errors}")
        self.logger.info(f"CACHE: last_complete {self.isofmt(self.last_complete, now_pc, now_dt)}")
        self.logger.info(f"CACHE: last_check {self.isofmt(self.last_check, now_pc, now_dt)}")
