# Release Engineering Tools

**IF YOU ARE NOT TRYING TO DO AN AMBASSADOR RELEASE, YOU'RE READING
THE WRONG DOCUMENT.** The release checklist lives at
https://www.notion.so/datawire/Release-Checklist-luke-v4-3d4006cc27ca4146a6ad47e5ad972a52

## Design statement

These tools help automate some of the tedious chunks of the above
checklist.  With the exception of the `lib/uiutil.py` and
`lib/ansiterm.py` utility libraries, the code should read as a trivial
translation of a human prose checklist in to Python.

For example, given the checklist item:

> - [ ] In `ambassador/docs/js/aes-pages.yml`, change `/pre-release/`
>   to `/latest/`.

The script `rel-01-rc-update-tree` has that translated as:

> ```python3
>     # apro.git/ambassador/docs/js/aes-pages.yml
>     for line in fileinput.FileInput("ambassador/docs/js/aes-pages.yml", inplace=True):
>         line = line.replace('/pre-release/', '/latest/')
>         sys.stdout.write(line)
>     git_add("ambassador/docs/js/aes-pages.yml")
> ```

The release process is maintained both as these tools, and as a human
checklist, so that if there is a bug or problem with the tools, then
there's a checklist for the human release engineer to fall back to, so
they don't need to try to debug the scripts while doing a release.

## Developing

Run `make check` to run typecheckers/linters/formatters to make sure
your code is good and is in a consistent style with the existing code.
