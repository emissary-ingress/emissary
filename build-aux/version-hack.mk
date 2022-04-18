# XXX THIS FILE IS A BRUTAL HACK XXX
#
# There are a bunch of files that used to get checked in to Git with
# version numbers in them.  This was terrible, as the version number
# changes every commit, so the files in Git are by definition wrong
# except for commits that are tagged as a release.  This was extra
# terrible because various things would edit those files, making the
# tree dirty.
#
# Getting away from that is difficult because the build system is a
# mess, and a bunch of things depend on those files without declaring
# dependencies on them, and updating everything all at once is not
# really viable.
#
# So, as a stepping stone to fixing that, don't check those files in
# to Git, but generate them early during Makefile parse-time (*before*
# Makefile execution-time), so that they are always available and
# up-to-date.  We do that by having a generated
# `version-hack.stamp.mk` Makefile-fragment depend on them, and
# `-include`ing that Makefile fragment.

#
# Normal recipes...

# These recipes themselves aren't brutal hacks, but before moving them
# to a less brutal-hack-y file, we should revisit if they should even
# exist.

# Hack: To add a layer to the hack, these recipes all start with '@'
# to prevent Make from printing the commands it's running.  Most of
# the time, DON'T DO THAT, IT MAKES THINGS HARD TO DEBUG!  But in this
# case, because Make runs them every parse, it would be very very
# noisy if Make actually printed them.

#
# Trigger Make to update those

build-aux/version-hack.stamp.mk: $(tools/write-ifchanged)
	@ls -l $^ | sed 's/^/#/' | $(tools/write-ifchanged) $@
# The "-include" directive (compared to plain "include") considers it
# non-fatal if the file can't be included (because it doesn't exist
# yet).  Then, when Make sees that it doesn't exist yet, it will say
# "oh, but I know how to create it", and then create it, and then it
# will restart itself in order to re-parse everything with the newly
# created file.
-include build-aux/version-hack.stamp.mk
