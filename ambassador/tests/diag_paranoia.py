import sys

import difflib
import json
import os

from shell import shell

from ambassador.config import Config

def prettify(obj):
    return json.dumps(obj, indent=4, sort_keys=True)

def mark_referenced_by(obj, refby):
    if '_referenced_by' not in obj:
        return True

    if refby not in obj['_referenced_by']:
        obj['_referenced_by'].append(refby)
        obj['_referenced_by'].sort()

        return True
    else:
        return False

Uniqifiers = {
    'breakers': lambda x: x['name'],
    'outliers': lambda x: x['name'],
    'filters': lambda x: x['name'],
    'tls': lambda x: "TLS",
    'listeners': lambda x: '%s-%s' % (x['service_port'], x['admin_port']),
    'routes': lambda x: x['group_id'],
    'sources': lambda x: '%s.%d' % (x['filename'], x['index']) if ('index' in x) else x['filename']
}

def filtered_overview(ov):
    filtered = {}

    for key in ov.keys():
        if not ov[key]:
            continue

        uniqifier = Uniqifiers.get(key, lambda x: x.get('name', None))

        filtered_element = []

        if isinstance(ov[key], list):
            for obj in ov[key]:
                if obj.get('_source', None) == '--internal--':
                    continue

                if '_referenced_by' in obj:
                    obj['_referenced_by'] = sorted([ x for x in obj['_referenced_by'] if x != '--internal--' ])

                filtered_element.append(obj)

            filtered[key] = sorted(filtered_element, key=uniqifier)
        else:
            # Make this a single-element list to match the reconstition.
            obj = ov[key]

            if '_referenced_by' in obj:
                obj['_referenced_by'].sort()

            filtered[key] = [ obj ]

    return filtered

def diag_paranoia(configdir, outputdir):
    aconf = Config(configdir)
    ov = aconf.diagnostic_overview()

    # print("==== OV")
    # print(prettify(ov))

    reconstituted = {}
    errors = []
    warnings = []
    missing_uniqifiers = {}

    for source in ov['sources']:
        source_filename = source['filename']

        for source_key in source['objects'].keys():
            intermediate = aconf.get_intermediate_for(source_key)

            # print("==== %s" % source_key)
            # print(prettify(intermediate))

            for key in intermediate.keys():
                if key == 'clusters':
                    rclusters = reconstituted.setdefault('clusters', {})

                    for cluster in intermediate[key]:
                        cname = cluster['name']
                        csource = cluster['_source']

                        if cname not in rclusters:
                            rclusters[cname] = dict(**cluster)
                            rclusters[cname]['_referenced_by'] = [ source_key ]

                            # print("%s: new cluster %s" % (source_key, prettify(rclusters[cname])))
                        else:
                            rcluster = rclusters[cname]
                            # print("%s: extant cluster %s" % (source_key, prettify(rclusters[cname])))

                            if not mark_referenced_by(rcluster, source_key):
                                errors.append('%s: already appears in cluster %s?' %
                                              (source_key, rcluster['name']))

                            for ckey in sorted(cluster.keys()):
                                if ckey == '_referenced_by':
                                    continue

                                if cluster[ckey] != rcluster[ckey]:
                                    errors.append("%s: cluster %s doesn't match %s for %s" % 
                                                  (source_key, cname, rcluster['_source'], ckey))

                            for rkey in sorted(rcluster.keys()):
                                if rkey not in cluster:
                                    errors.append('%s: cluster %s is missing key %s from source %s' % 
                                                  (source_key, cname, rkey, rcluster['_source']))
                else:
                    # Other things are a touch more straightforward, just need to work out a unique
                    # key for them.

                    uniqifier = Uniqifiers.get(key, None)

                    if not uniqifier:
                        if not key in missing_uniqifiers:
                            warnings.append("missing uniqifier for %s" % key)
                            missing_uniqifiers[key] = True
                        continue

                    for obj in intermediate[key]:
                        u = uniqifier(obj)

                        rcon = reconstituted.setdefault(key, {})

                        if u in rcon:
                            if obj['_source'] != rcon[u]['_source']:
                                errors.append('%s: %s %s already defined by %s' %
                                              (source_key, key, u, prettify(rcon[u])))
                            else:
                                mark_referenced_by(rcon[u], obj['_source'])
                        else:
                            rcon[u] = obj

                            if '_referenced_by' in rcon[u]:
                                rcon[u]['_referenced_by'].sort()

    # OK. After all that, flip the dictionaries in reconstituted back into lists...

    reconstituted_lists = {}

    for key in reconstituted:
        if key == 'sources':
            # Special work here: reassemble source files from objects.
            sources = {}

            for source_key, obj in reconstituted['sources'].items():
                # print(obj)
                s = sources.setdefault(obj['filename'], {
                    'count': 0,
                    'error_count': 0,
                    'filename': obj['filename'],
                    'objects': {}
                })

                s['count'] += 1
                s['objects'][source_key] = {
                    'errors': obj['errors'],
                    'key': source_key,
                    'kind': obj['kind']
                }
                s['error_count'] += len(obj['errors'])

            for s in sources.values():
                s['error_plural'] = "error" if (s['error_count'] == 1) else "errors"
                s['plural'] = "object" if (s['count'] == 1) else "objects"

            # Finally, sort 'em all.
            reconstituted_lists['sources'] = sorted(sources.values(), key=lambda x: x['filename'])
        else:
            # Not the list of sources. Grab the uniqifier...
            uniqifier = Uniqifiers.get(key, lambda x: x.get('name', None))

            reconstituted_lists[key] = sorted(reconstituted[key].values(), key=uniqifier)

    # If there's no listener block in the reconstituted set, that implies that 
    # the configuration doesn't override the listener state. Go ahead and add the
    # default in.

    if 'listeners' not in reconstituted_lists:
        reconstituted_lists['listeners'] = [
            {
                "_source": "--internal--",
                "admin_port": 8001,
                "service_port": 80
            }
        ]

    # If there're no 'filters' in the reconstituted set, uh, there were no filters
    # defined. Create an empty list.

    if 'filters' not in reconstituted_lists:
        reconstituted_lists['filters'] = []

    # OK. Next, filter out the '--internal--' stuff from our overview, and sort
    # _referenced_by.
    filtered = filtered_overview(ov)

    pretty_filtered_overview = prettify(filtered)
    pretty_reconstituted_lists = prettify(reconstituted_lists)

    udiff = list(difflib.unified_diff(pretty_filtered_overview.split("\n"),
                                      pretty_reconstituted_lists.split("\n"),
                                      fromfile="from overview", tofile="from intermediates",
                                      lineterm=""))

    if udiff:
        errors.append("%s\n-- DIFF --\n%s\n-- OVERVIEW --\n%s\n\n-- RECONSTITUTED --\n%s\n" %
                      ("mismatch between overview and reconstituted diagnostics",
                       "\n".join(udiff), pretty_filtered_overview, pretty_reconstituted_lists))

    return {
        'errors': errors,
        'warnings': warnings
    }

if __name__ == "__main__":
    results = diag_paranoia(sys.argv[1], ".")

    if (results['warnings']):
        print("\n".join(['WARNING: %s' % x for x in results['warnings']]))

    if (results['errors']):
        print("\n".join(['ERROR: %s' % x for x in results['errors']]))
        sys.exit(1)
    else:
        sys.exit(0)
