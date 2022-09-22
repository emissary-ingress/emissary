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

from multi import multi


def test_when():
    @multi
    def area(record):
        return record["shape"]

    @area.when("square")
    def area(square):
        return square["width"] * square["height"]

    @area.when("triangle")
    def area(tri):
        return 0.5 * tri["base"] * tri["height"]

    assert area({"shape": "square", "width": 10, "height": 10}) == 100
    assert area({"shape": "triangle", "base": 10, "height": 10}) == 50

    try:
        area({"shape": "hourglass"})
        assert False, "should fail"
    except TypeError as e:
        assert (
            str(e)
            == "no match found for multi function area({'shape': 'hourglass'}): known keys ('square', 'triangle'), searched keys ('hourglass',)"
        )


def test_default():
    @multi
    def fmt(obj):
        return type(obj)

    @fmt.when(tuple)
    def fmt(tup):
        return ", ".join(fmt(x) for x in tup)

    @fmt.when(dict)
    def fmt(d):
        return ", ".join("%s=%s" % (fmt(k), fmt(v)) for k, v in d.items())

    @fmt.when(str)
    def fmt(s):
        return s

    @fmt.when(int)
    def fmt(i):
        return str(i)

    @fmt.when(float)
    def fmt(f):
        return "%.2f" % f

    @fmt.default
    def fmt(o):
        return "fmt(%r)" % o

    assert fmt((1, 2, 3)) == "1, 2, 3"
    assert fmt({"x": "y"}) == "x=y"
    assert fmt("asdf") == "asdf"
    assert fmt(3) == "3"
    assert fmt(3.14159265359) == "3.14"
    assert fmt([1, 2, 3]) == "fmt(%r)" % [1, 2, 3]


def test_multiple_keys():
    @multi
    def fib(x):
        return x

    @fib.when(0, 1)
    def fib(x):
        return x

    @fib.default
    def fib(x):
        return fib(x - 1) + fib(x - 2)

    assert fib(0) == 0
    assert fib(1) == 1
    assert fib(2) == 1
    assert fib(3) == 2
    assert fib(4) == 3
    assert fib(5) == 5


def test_generator():
    @multi
    def fib(x):
        yield x
        yield type(x)

    @fib.when(0, 1)
    def fib(x):
        return x

    @fib.when(int)
    def fib(x):
        return fib(x - 1) + fib(x - 2)

    assert fib(0) == 0
    assert fib(1) == 1
    assert fib(2) == 1
    assert fib(3) == 2
    assert fib(4) == 3
    assert fib(5) == 5


def test_method():
    class Foo:
        def __init__(self):
            pass

        @multi
        def fib(self, x):
            return x

        @fib.when(0, 1)
        def fib(self, x):
            return x

        @fib.default
        def fib(self, x):
            return fib(x - 1) + fib(x - 2)

    fib = Foo().fib

    assert fib(0) == 0
    assert fib(1) == 1
    assert fib(2) == 1
    assert fib(3) == 2
    assert fib(4) == 3
    assert fib(5) == 5


def test_init():
    class Foo:
        @multi
        def __init__(self, x):
            return x

        @__init__.when(0, 1)
        def __init__(self, x):
            self.x = x

        @__init__.default
        def __init__(self, x):
            self.x = 2 * x

    assert Foo(0).x == 0
    assert Foo(1).x == 1
    assert Foo(2).x == 4


def test_do():
    @multi
    def foo(x):
        return x

    @foo.when("x")
    def do(x):
        return "y"

    assert foo("x") == "y"
