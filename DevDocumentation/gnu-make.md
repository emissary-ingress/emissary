A quick primer on GNU Make syntax
=================================

This tries to cover the syntax that is hard to ctrl-f for in
<https://www.gnu.org/software/make/manual/make.html> (err, hard to
C-s for in `M-: (info "Make")`).

  At the core is a "rule":

      target: dependency1 dependency2
      	command to run

  If `target` something that isn't a real file (like 'build', 'lint', or
  'test'), then it should be marked as "phony":

      target: dependency1 dependency2
      	command to run
      .PHONY: target

  You can write reusable "pattern" rules:

      %.o: %.c
      	command to run

  Of course, if you don't have variables for the inputs and outputs,
  it's hard to write a "command to run" for a pattern rule.  The
  variables that you should know are:

      $@ = the target
      $^ = the list of dependencies (space separated)
      $< = the first (left-most) dependency
      $* = the value of the % glob in a pattern rule

      Each of these have $(@D) and $(@F) variants that are the
      directory-part and file-part of each value, respectively.

      I think those are easy enough to remember mnemonically:
        - $@ is where you shoul direct the output at.
        - $^ points up at the dependency list
        - $< points at the left-most member of the dependency list
        - $* is the % glob; "*" is well-known as the glob char in other languages

  Make will do its best to guess whether to apply a pattern rule for a
  given file.  Or, you can explicitly tell it by using a 3-field
  (2-colon) version:

      foo.o bar.o: %.o: %.c
      	command to run

  In a non-pattern rule, if there are multiple targets listed, then it
  is as if rule were duplicated for each target:

      target1 target2: deps
      	command to run

      # is the same as

      target1: deps
      	command to run
      target2: deps
      	command to run

  Because of this, if you have a command that generates multiple,
  outputs, it _must_ be a pattern rule:

      %.c %.h: %.y
      	command to run

  Normally, Make crawls the entire tree of dependencies, updating a file
  if any of its dependencies have been updated.  There's a really poorly
  named feature called "order-only" dependencies:

      target: normal-deps | order-only-deps

  Dependencies after the "|" are created if they don't exist, but if
  they already exist, then don't bother updating them.

Tips:
-----

 - Use absolute filenames.  It's dumb, but it really does result in
   fewer headaches.  Use $(OSS_HOME) and $(AES_HOME) to spell the
   absolute filenames.

 - If you have a multiple-output command where the output files have
   dissimilar names, have % be just the directory (the above tip makes
   this easier).

 - It can be useful to use the 2-colon form of a pattern rule when
   writing a rule for just one file; it lets you use % and $* to avoid
   repeating yourself, which can be especially useful with long
   filenames.
