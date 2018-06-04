#!python

import sys

import re
import yaml

import dpath

class Edit (object):
    def __init__(self):
        self.elements = {}

    def __nonzero__(self):
        return bool(self.elements)

    def __str__(self):
        return yaml.safe_dump(self.elements, default_flow_style=False)

    def add(self, element, args):
        self.elements.setdefault(element.lower(), []).append(args)

    def match(self, manifest, x):
        all_matched = True

        if 'match' in self.elements:
            for path, value in self.elements['match']:
                qualified = '/%d/%s' % (x, path)
                got_value = dpath.util.get(manifest, qualified)

                if got_value != value:
                    print("NOMATCH %s: %s != %s" % (qualified, value, got_value))
                    all_matched = False
                else:
                    print("MATCH %s == %s" % (qualified, value))

        return all_matched

    def set(self, manifest, x, path, value):
        qualified = '/%d/%s' % (x, path)
        print("SET %s = %s" % (qualified, value))
        dpath.util.new(manifest, qualified, value)

    def execute(self, manifest, x, args):
        if 'discard' in self.elements:
            print("DISCARD")
            return False

        for element in self.elements.get('mklist', []):
            path = element[0]
            self.set(manifest, x, path, [])

        for element in self.elements.get('mkdict', []):
            path = element[0]
            self.set(manifest, x, path, {})

        for element in self.elements.get('set', []):
            path, value = element
            self.set(manifest, x, path, args.interpolate(value))

        for element in self.elements.get('setint', []):
            path, value = element
            self.set(manifest, x, path, int(value))

        for element in self.elements.get('delete', []):
            path = element[0]
            qualified = '/%d/%s' % (x, path)
            print("DEL %s" % qualified)
            dpath.util.delete(manifest, qualified)

        return True

class Edits (object):
    def __init__(self):
        self.edits = []
        self.current = Edit()

    def __iter__(self):
        return iter(self.edits)

    def __str__(self):
        output = ''

        for edit in edits:
            output += '---\n%s\n' % edit

        return output

    def add(self, element, args):
        self.current.add(element, args)

    def finish(self):
        if self.current:
            self.edits.append(self.current)
            self.current = Edit()

class Args (object):
    reVar = re.compile(r'\$(\d+)')

    def __init__(self):
        self.needed = 0
        self.args = None

    def load(self, args):
        if len(args) != self.needed:
            raise Exception("need %d arg%s, have %d\n" % 
                            (self.needed, "" if self.needed == 1 else "s", 
                             len(args)))

        self.args = args

    def counter(self, matchobj):
        var = int(matchobj.group(1))

        if var > self.needed:
            self.needed = var

    def replacer(self, matchobj):
        var = int(matchobj.group(1))

        return self.args[var - 1]

    def count(self, args):
        for text in args:
            Args.reVar.sub(self.counter, text)

    def interpolate(self, text):
        return Args.reVar.sub(self.replacer, text)

cmd_path = sys.argv.pop(1)  # Not 0. Trust me.

edits = Edits()
args = Args()

for line in open(cmd_path, 'r'):
    line = line.strip()

    if not line:
        edits.finish()
        continue

    fields = line.split(' ')

    element = fields[0]
    element_args = fields[1:]

    args.count(element_args)    
    edits.add(element, element_args)

edits.finish()

print(edits)
print("args needed: %s" % args.needed)

input_yaml_path = "-"
output_yaml_path = "-"

if len(sys.argv) > 1:
    input_yaml_path = sys.argv.pop(1)

    if len(sys.argv) > 1:
        output_yaml_path = sys.argv.pop(1)

args.load(sys.argv[1:])

input_yaml = sys.stdin if input_yaml_path == "-" else open(input_yaml_path, 'r')
output_yaml = sys.stdin if output_yaml_path == "-" else open(output_yaml_path, 'w')
    
manifest = list(yaml.safe_load_all(input_yaml))

keep = []

for x in range(len(manifest)):
    matched = False

    for edit in edits:
        if edit.match(manifest, x):
            matched = True

            if edit.execute(manifest, x, args):
                keep.append(manifest[x])
            
            break
    
    if not matched:
        keep.append(manifest[x])

yaml.safe_dump_all(keep, output_yaml, default_flow_style=False)
