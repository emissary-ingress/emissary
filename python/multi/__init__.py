# Copyright 2018 Datawire. All rights reserved.
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

# This is based on the blog post described here, but with a few
# differences noted below:
#
#   https://adambard.com/blog/implementing-multimethods-in-python/
#
# The differences are:
#
# 1. The "default default" action is to raise a TypeError rather than
#    to be a noop. This avoids many classes of silent failure.
#
# 2. The naming is a bit different:
#    a) the initial decorator is the same:
#
#       @multi
#       def foo(...): ...
#
#    b) rather than adding actions with @method(foo, case), this uses @foo.when(case)
#    c) rather than specifying the default action with @method(foo), this uses @foo.default
#
# 3. You can specify multiple cases at once, e.g.:
#
#      @foo.when(a, b, c)
#      def foo(...): ...
#
# 4. If foo is a generator then the dispatch logic will check each key
#    yielded in turn until a result is found, this allows for more
#    flexible dispatch logic like this:
#
#      @multi
#      def fib(x):
#          yield x        # first dispatch on the value of x itself
#          yield type(x)  # if there are no matches, then dispatch on the type of x
#
#      @fib.when(0, 1)
#      def fib(x):
#          return x
#
#      @fib.when(int)
#      def fib(x):
#          return fib(x-1) + fib(x-2)

import functools, inspect


def _error(multifun, keys, args, kwargs):
    sargs = [repr(a) for a in args] + ["%s=%r" for k, v in kwargs.items()]
    raise TypeError(
        "no match found for multi function %s(%s): known keys %r, searched keys %r"
        % (multifun.__name__, ", ".join(sargs), tuple(multifun.__multi__.keys()), tuple(keys))
    )


def multi(dispatch_fn):
    gen = inspect.isgeneratorfunction(dispatch_fn)

    if gen:

        def multifun(*args, **kwargs):
            for key in dispatch_fn(*args, **kwargs):
                try:
                    action = multifun.__multi__[key]
                    break
                except KeyError:
                    continue
            else:
                action = multifun.__multi_default__
            return action(*args, **kwargs)

    else:

        def multifun(*args, **kwargs):
            key = dispatch_fn(*args, **kwargs)
            action = multifun.__multi__.get(key, multifun.__multi_default__)
            return action(*args, **kwargs)

    multifun.when = lambda *keys: _when(multifun, keys)
    multifun.default = _default(multifun)
    multifun.__multi__ = {}
    # Default default
    multifun.__multi_default__ = lambda *args, **kwargs: _error(
        multifun,
        dispatch_fn(*args, **kwargs) if gen else [dispatch_fn(*args, **kwargs)],
        args,
        kwargs,
    )

    functools.update_wrapper(multifun, dispatch_fn)
    return multifun


def _when(multifun, keys):
    def apply_decorator(action):
        for k in keys:
            multifun.__multi__[k] = action
        return multifun

    return apply_decorator


def _default(multifun):
    def apply_decorator(action):
        multifun.__multi_default__ = action
        return multifun

    return apply_decorator
