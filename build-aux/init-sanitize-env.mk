# Sanitize the environment a bit.
unexport ENV      # bad configuration mechanism
unexport BASH_ENV # bad configuration mechanism, but CircleCI insists on it
unexport CDPATH   # should not be exported, but some people do
unexport IFS      # should not be exported, but some people do

# In the days before Bash 2.05 (April 2001), Bash had a hack in it
# where it would load the interactive-shell configuration when run
# from sshd, I guess to work around buggy sshd implementations that
# didn't run the shell as login or interactive or something like that.
# But that hack was removed in Bash 2.05 in 2001.  And the changelog
# indicates that the heuristics it used to decide whether to do that
# were buggy to begin with, and it would often trigger when it
# shouldn't.  BUT DEBIAN PATCHES BASH TO ADD THAT HACK BACK IN!  And,
# more importantly, Ubuntu 20.04 (which our CircleCI uses) inherits
# that patch from Debian.  And the heuristic that Bash uses
# incorrectly triggers inside of Make in our CircleCI jobs!  So, unset
# SSH_CLIENT and SSH2_CLIENT to disable that.
unexport SSH_CLIENT
unexport SSH2_CLIENT
