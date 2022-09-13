# Copyright 2021 Datawire. All rights reserved.
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

import logging
import os
from typing import Any, Dict

from ambassador.utils import parse_bool

######
# Utilities for hostglob_matches
#
# hostglob_matches_start has g1 starting with '*' and g2 not ending with '*';
# it's OK for g2 to start with a wilcard too.


def hostglob_matches_start(g1: str, g2: str, g2start: bool) -> bool:
    # Leading "*" cannot match an empty string, so unless we have a wildcard
    # for g2, we have to have g1 longer than g2.

    g1match = g1[1:]
    g2match = g2[1:] if g2start else g2

    if len(g1) > len(g2match):
        if not g2start:
            # logging.debug("  match start: %s is too short => False", g1)
            return False

        # Wildcards for both, so make sure we do the substring match against
        # the longer one.
        g1match = g2[1:]
        g2match = g1[1:]

    match = g2match.endswith(g1match)
    # logging.debug("  match start: %s ~ %s => %s", g1match, g2match, match)

    return match


# hostglob_matches_end has g1 ending with '*' and g2 not starting with '*';
# it's OK for g2 to end with a wilcard too.


def hostglob_matches_end(g1: str, g2: str, g2end: bool) -> bool:
    # Leading "*" cannot match an empty string, so unless we have a wildcard
    # for g2, we have to have g1 longer than g2.
    g1match = g1[:-1]
    g2match = g2[:-1] if g2end else g2

    if len(g1) > len(g2match):
        if not g2end:
            # logging.debug("  match end: %s is too short => False", g1)
            return False

        # Wildcards for both, so make sure we do the substring match against
        # the longer one.
        g1match = g2[:-1]
        g2match = g1[:-1]

    match = g2match.startswith(g1match)
    # logging.debug("  match end: %s ~ %s => %s", g1match, g2match, match)

    return match


################


def hostglob_matches(g1: str, g2: str) -> bool:
    """
    hostglob_matches determines whether or not two given DNS globs are
    compatible with each other, i.e. whether or not there can be a hostname
    that matches both globs.

    Note that it does not actually find such a hostname: a return of True
    just means that such a hostname could exist.
    """

    # logging.debug("hostglob_matches: %s ~ %s", g1, g2)

    # Short-circuit: if g1 & g2 are equal, we're done here.
    if g1 == g2:
        # logging.debug("  equal => True")
        return True

    # Next special case: if either glob is "*", then it matches everything.
    if (g1 == "*") or (g2 == "*"):
        # logging.debug("  \"*\" present => True")
        return True

    # Final special case: if either starts with a bare ".", that's not OK.
    # (Ending with a bare "." is different because DNS.)
    if g1[0] == "." or g2[0] == ".":
        # logging.debug("  exact match starts with bare \".\" => False")
        return False

    # OK, we don't have the simple-"*" case, so any wildcards must be at
    # the start or end, and they must be a component alone.
    g1start = g1[0] == "*"
    g1end = g1[-1] == "*"
    g2start = g2[0] == "*"
    g2end = g2[-1] == "*"

    # logging.debug("  g1start=%s g1end=%s g2start=%s g2end=%s", g1start, g1end, g2start, g2end)

    if (g1start and g1end) or (g2start and g2end):
        # Not a valid DNS glob: you can't have a "*" at both ends. (If you do,
        # Envoy will decide that the one at the start is the allowed wildcard, and
        # treat the one at the end as a literal "*", which will match nothing.)
        return g1 == g2

    if not (g1start or g1end or g2start or g2end):
        # No valid wildcards. and we already know that they're not equal,
        # so this is not a match.
        # logging.debug("  not equal => False")
        return False

    # OK, if we're here, we have a wildcard to check. There are a few cases
    # here, so we'll start with the easy one: one value starts with "*" and
    # the other ends with "*", because those can always overlap as long as
    # the overlap between isn't empty -- and in this method, we only need to
    # concern ourselves with being sure that there is a possibility of a match
    # to both.
    if (g1start and g2end) or (g2start and g1end):
        # logging.debug("  start/end pair => True")
        return True

    # OK, now we have to actually do some work. Again, we really only have to
    # be convinced that it's possible for something to match, so e.g.
    #
    # *example.com, example.com
    #
    # is not a valid pair, because that "*" must never match an empty string.
    # However,
    #
    # *example.com, *.example.com
    #
    # is fine, because e.g. "foo.example.com" matches both.

    if g1start:
        return hostglob_matches_start(g1, g2, g2start)

    if g2start:
        return hostglob_matches_start(g2, g1, g1start)

    if g1end:
        return hostglob_matches_end(g1, g2, g2end)

    if g2end:
        return hostglob_matches_end(g2, g1, g1end)

    # This is "impossible"
    return False


################
## selector_matches is a utility for doing K8s label selector matching.


def selector_matches(
    logger: logging.Logger, selector: Dict[str, Any], labels: Dict[str, str]
) -> bool:
    match: Dict[str, str] = selector.get("matchLabels") or {}

    if not match:
        # If there's no matchLabels to match, return True.
        logger.debug("    no matchLabels in selector => True")
        return True

    # If we have stuff to match on, but no labels to actually match them, we
    # can short-circuit (and skip a weirder conditional down in the loop).
    if not labels:
        logger.debug("    no incoming labels => False")
        return False

    # Ambassador (2.0-2.3) & (3.0-3.1) consider a match on a single label as a "good enough" match.
    # In versions 2.4+ and 3.2+ _ALL_ labels in a selector must be present for it to be considered a match.
    # DISABLE_STRICT_LABEL_SELECTORS provides a way to restore the old unintended loose matching behaviour
    # in the event that it is desired. The ability to disable strict label matching will be removed in a future version.
    disable_strict_selectors = parse_bool(os.environ.get("DISABLE_STRICT_LABEL_SELECTORS", "false"))

    # For every label in mappingSelector, there must be a label with same value in Mapping itself.
    for k, v in match.items():
        if labels.get(k) == v:
            logger.debug("    selector match for %s=%s => True", k, v)
            if disable_strict_selectors:
                return True
        elif not disable_strict_selectors:
            logger.debug("    selector miss for %s=%s => False", k, v)
            return False

    logger.debug(f"    all selectors match => True")
    return True
