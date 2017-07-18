#!/usr/bin/env python

import sys

import logging

import subprocess, re, os
from semantic_version import Version
from git import Repo

dry_run = True

class VersionedBranch (object):
    """ A branch for which we're going to wrangle versions based on tags. """

    def __init__(self, git_repo, git_head):
        """ Here git_repo and git_head are gitpython objects, not just strings. """
        self.log = logging.getLogger("VersionedBranch")

        self.repo = git_repo
        self.head = git_head
        self.branch_name = git_head.name

        self._version_tag = None
        self._version = None
        self._versioned_commit = None
        self._current_commit = None

    @property
    def version_tag(self):
        if self._version_tag is None:
            try:
                tag_name = self.repo.git.describe(self.branch_name, abbrev=0, match='v[0-9]*')

                self._version_tag = self.repo.tags[tag_name]

                self.log.debug("version_tag: %s => %s" % (self.branch_name, self._version_tag.name))
            except Exception as e:
                self.log.warning("version_tag: %s got no tag: %s" % (self.branch_name, e))

        return self._version_tag
    
    @property
    def version(self):
        if (self._version is None) and (self.version_tag is not None):
            self._version = Version(self.version_tag.name[1:])
            self.log.debug("version: %s => %s" % (self.branch_name, self._version))

        return self._version

    @property
    def versioned_commit(self):
        if (self._versioned_commit is None) and (self.version_tag is not None):
            self._versioned_commit = self._version_tag.commit
            self.log.debug("versioned_commit: %s => %s @ %s" % 
                           (self.branch_name, self.version, self._versioned_commit))

        return self._versioned_commit

    @property
    def current_commit(self):
        if not self._current_commit:
            self._current_commit = self.head.commit
            self.log.debug("current_commit: %s => %s" %
                           (self.branch_name, self._current_commit))

        return self._current_commit

    def __unicode__(self):
        return ("<VersionedBranch %s @ %s [%s @ %s]>" %
                (self.branch_name, str(self.current_commit)[0:8],
                 self.version, str(self.versioned_commit)[0:8]))

    def __str__(self):
        return unicode(self)

    def recent_commits(self, since_tag=None):
        if not since_tag:
            since_tag = self.version_tag.name

        for line in self.repo.git.log("--reverse", "--oneline", self.branch_name,
                                      "--not", since_tag).split("\n"):
            self.log.debug("line: %s" % line)

            if line:
                commitID, subject = line.split(" ", 1)

                yield commitID, subject

    def next_version(self, magic_pre=False, since_tag=None, reduced_zero=True, commit_map=None,
                     pre_release=None, build=None):
        rdelta = ReleaseDelta(self, magic_pre=magic_pre, 
                              since_tag=since_tag, reduced_zero=reduced_zero,
                              commit_map=commit_map, pre_release=pre_release, build=build)

        return rdelta.next_version

class VersionDelta(object):
    def __init__(self, scale, xform, tag):
        self.scale = scale
        self.xform = xform
        self.tag = tag
        self.delta = scale

    def __cmp__(self, other):
        if self.scale < other.scale:
            return -1
        elif self.scale > other.scale:
            return 1
        else:
            return 0

    def __lt__(self, other):
        return self.__cmp__(other) < 0

    def __gt__(self, other):
        return self.__cmp__(other) > 0

    def __unicode__(self):
        return "<VersionDelta %s>" % self.tag

    def __str__(self):
        return self.__unicode__()

class ReleaseDelta(object):
    FIX   = VersionDelta( (0,0,1), Version.next_patch, "[FIX]")
    MINOR = VersionDelta( (0,1,0), Version.next_minor, "[MINOR]")
    MAJOR = VersionDelta( (1,0,0), Version.next_major, "[MAJOR]")

    """ how new commits affect project version """
    log = logging.getLogger("ReleaseDelta")

    def __init__(self, vbr, magic_pre=False, since_tag=None,
                 reduced_zero=True, commit_map=None,
                 pre_release=None, build=None):
        self.vbr = vbr
        self.magic_pre = magic_pre
        self.since_tag = since_tag
        self.pre_release = pre_release
        self.build = build
        self.commit_map = commit_map

        if reduced_zero and (self.vbr.version.major == 0):
            self.log.debug("While the project is in %s version, all changes have reduced impact" % self.vbr.version)
            self.MAJOR.xform = self.MINOR.xform
            self.MAJOR.delta = self.MINOR.delta
            self.MINOR.xform = self.FIX.xform
            self.MINOR.delta = self.FIX.delta

    def commits(self):
        # for commit, subject in [
        #     ( '123456', '[MINOR]' )
        # ]:
        #     yield commit, subject

        for commit, subject in self.vbr.recent_commits(self.since_tag):
            if self.commit_map and (commit in self.commit_map):
                subject = self.commit_map[commit]

            yield commit, subject

    def commit_deltas(self):
        for commitID, subject in self.commits():
            delta = self.FIX
            source = "by default"

            for commitDelta in [self.MAJOR, self.MINOR, self.FIX]:
                if commitDelta.tag in subject:
                    delta = commitDelta
                    source = "from commit message"
                    break

            if source == "by default":
                if subject.strip().startswith('feat('):
                    delta = self.MINOR
                    source = "from feature marker"
                elif subject.strip().startswith('break:'):
                    delta = self.MAJOR
                    source = "from breakage marker"

            self.log.debug("commit %s: %s %s\n-- [%s]" % (commitID, delta.tag, source, subject))

            yield delta, commitID, subject

    def version_change(self):
        finalDelta = None
        commits = []

        for delta, commitID, subject in self.commit_deltas():
            self.log.debug("folding %s: %s" % (commitID, delta))

            commits.append((delta, commitID, subject))

            if finalDelta is None:
                self.log.debug("%s: initial change %s" % (commitID, delta))
                finalDelta = delta
            elif delta > finalDelta:
                self.log.debug("%s: %s overrides %s" % (commitID, delta, finalDelta))

                finalDelta = delta

        if not commits:
            self.log.debug("version_change: no commits since %s" % self.vbr.version)
            return None, None
        else:
            self.log.debug("folding %d commit%s into %s: delta %s" % 
                           (len(commits), "" if len(commits) == 1 else "s", 
                            finalDelta, finalDelta.delta))

            return finalDelta, commits

    @property
    def next_version(self):
        version = self.vbr.version
        self.log.debug("version start: %s" % version)

        finalDelta, commits = self.version_change()

        if finalDelta:
            self.log.debug("final commit list: %s" % 
                           "\n".join(map(lambda x: "%s %s: %s" % (x[0].tag, x[1], x[2]),
                                         commits)))
            self.log.debug("final change:      %s %s" % (finalDelta, finalDelta.delta))

            version = finalDelta.xform(version)

            self.log.debug("version:           %s" % version)

            if self.magic_pre:
                pre = self.vbr.version.prerelease

                self.log.debug("magic check:      '%s'" % str(pre))

                if pre:
                    pre = pre[0]

                    if pre and pre.startswith('b'):
                        if finalDelta > self.FIX:
                            pre = "b1"
                        else:
                            pre = "b" + str(int(pre[1:]) + 1)

                    self.log.debug("magic prerelease:  %s" % str(pre))
                    version.prerelease = (pre,)
                else:
                    version.prerelease = ("b1",)
            elif self.pre_release:
                version.prerelease = (self.pre_release,)

            self.log.debug("final prerelease:  %s" % str(version.prerelease))

            if self.build:
                version.build = (self.build,)
                self.log.debug("final build:       %s" % str(version.build))

            self.log.debug("version has to change from %s to %s" %
                           (self.vbr.version, version))

            return version
        else:
            return None

class VersionedRepo (object):
    """ Representation of a git repo that follows our versioning rules """

    def __init__(self, repo_root):
        self.log = logging.getLogger("VersionedRepo")

        self.repo = Repo(repo_root, search_parent_directories=True)

        self.branches = {}

    def get_branch(self, branch_name):
        # Grab a branch of this repo

        key = branch_name if branch_name else '*current*'
        source = 'cache'
        vbr = None

        if key in self.branches:
            vbr = self.branches[key]

        if not vbr:
            source = 'Git'

            head = None

            if branch_name:
                print(self.repo.heads)

                head = self.repo.heads[branch_name]
            else:
                # head = self.repo.active_branch
                head = self.repo.head                

            if not head:
                self.log.warning("get_branch: no branch %s" % branch_name)

            vbr = VersionedBranch(self.repo, head)

            self.branches[key] = vbr

        self.log.debug("get_branch: got %s from %s" % (key, source))

        return vbr

    def tag_version(self, version, commit):
        tag_name = str(version)

        new_tag = self.repo.create_tag(tag_name, commit)

        return new_tag

if __name__ == '__main__':
    from docopt import docopt

    __doc__ = """versioner.py

    Manipulate version tags

    Usage: 
        versioner.py [-n] [--verbose] [options]

    Options:
        --branch=<branchname>      set which branch to work on
        --magic-pre                do magic autoincrementing prerelease numbers
        --pre=<pre-release-tag>    explicitly set the prerelease number
        --build=<build-tag>        explicitly set the build number
        --since=<since-tag>        override the tag of the last release
    """

    args = docopt(__doc__, version="versioner {0}".format("0.1.0"))

    dryrun = args["-n"]

    level = logging.INFO

    if args["--verbose"]:
        level = logging.DEBUG

    logging.basicConfig(level=level)

    vr = VersionedRepo(os.getcwd())
    vbr = vr.get_branch(args.get('--branch', None))

    # print(vbr)

    # commit_map = {
    #     'e787009': '[MAJOR] OH GOD NO',
    #     'a889b73': '[MINOR] Oh yeah!',
    #     '2d0b5ec': '[MINOR] WTFO?'
    # }

    next_version = vbr.next_version(magic_pre=args.get('--magic-pre', False),
                               pre_release=args.get('--pre', None),
                               build=args.get('--build', None),
                               since_tag=args.get('--since', None),
                               reduced_zero=False)
    #                           commit_map=commit_map)

    if next_version:
        print(next_version)
    else:
        sys.stderr.write("no changes since %s\n" % vbr.version)
        sys.exit(1)
