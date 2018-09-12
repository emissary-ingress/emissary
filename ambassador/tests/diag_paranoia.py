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
    'admin': lambda x: x['admin_port'],
    'breakers': lambda x: x['name'],
    'clusters': lambda x: ( x['_source'], x['name'] ),
    'outliers': lambda x: x['name'],
    'filters': lambda x: x['name'],
    'grpc_services': lambda x: x['name'],
    'tls': lambda x: "TLS",
    # 'tls': lambda x: "cors_default",
    'listeners': lambda x: '%s-%s' % (x['service_port'], x.get('require_tls', False)),
    'routes': lambda x: x['_group_id'],
    'sources': lambda x: '%s.%d' % (x['filename'], x['index']) if (('index' in x) and (x['filename'] != "--internal--")) else x['filename'],
    'tracing': lambda x: x['cluster_name']
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
                # if obj.get('_source', None) == '--internal--':
                #     continue

                if '_referenced_by' in obj:
                    obj['_referenced_by'] = sorted([ x for x in obj['_referenced_by'] ])

                filtered_element.append(obj)

            filtered[key] = sorted(filtered_element, key=uniqifier)
        else:
            # Make this a single-element list to match the reconstition.
            obj = ov[key]

            if '_referenced_by' in obj:
                obj['_referenced_by'].sort()

            filtered[key] = [ obj ]

    return filtered

def sanitize_errors(ov):
    sources = ov.get('sources', {})
    errors = ov.get('errors', {})
    filtered = {}

    for key in errors.keys():
        errlist = errors[key]
        filtlist = []

        for error in errlist:
            error['version'] = 'sanitized'
            error['hostname'] = 'sanitized'
            error['resolvedname'] = 'sanitized'

            filtlist.append(error)

        filtered[key] = filtlist

        if key in sources:
            sources[key]['errors'] = filtlist

    return ov

def diag_paranoia(configdir, outputdir):
    aconf = Config(configdir)
    ov = aconf.diagnostic_overview()

    reconstituted = {}
    errors = []
    warnings = []
    missing_uniqifiers = {}

    source_info = [
        {
            "filename": x['filename'],
            "sources": [ key for key in x['objects'].keys() ]
        }
        for x in ov['sources']
    ]

    source_info.insert(0, {
        "filename": "--internal--",
        "sources": [ "--internal--" ]
    })

    for si in source_info:
        source_filename = si['filename']

        for source_key in si['sources']:
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

                            if not mark_referenced_by(rcluster, source_key) and (source_key != "--internal--"):
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
                        # print(obj)
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
                    '_errors': obj['_errors'],
                    'key': source_key,
                    'kind': obj['kind']
                }
                s['error_count'] += len(obj['_errors'])

            for s in sources.values():
                s['error_plural'] = "error" if (s['error_count'] == 1) else "_errors"
                s['plural'] = "object" if (s['count'] == 1) else "objects"

            # Finally, sort 'em all.
            reconstituted_lists['sources'] = sorted(sources.values(), key=lambda x: x['filename'])
        else:
            # Not the list of sources. Grab the uniqifier...
            uniqifier = Uniqifiers.get(key, lambda x: x.get('name', None))

            reconstituted_lists[key] = sorted(reconstituted[key].values(), key=uniqifier)

    # # If there's no listener block in the reconstituted set, that implies that 
    # # the configuration doesn't override the listener state. Go ahead and add the
    # # default in.

    # l = reconstituted_lists.get('listeners', [])

    # if not l:
    #     reconstituted_lists['listeners'] = []

    # If there're no 'filters' in the reconstituted set, uh, there were no filters
    # defined. Create an empty list.

    if 'filters' not in reconstituted_lists:
        reconstituted_lists['filters'] = []

    # Copy any 'ambassador_services' block from the original into the reconstituted list.
    if ('ambassador_services' in ov) and ('ambassador_services' not in reconstituted_lists):
        reconstituted_lists['ambassador_services'] = ov['ambassador_services']

    # Copy any 'cors_default_envoy' block from the original into the reconstituted list.
    if ('cors_default' in ov) and ('cors_default' not in reconstituted_lists):
        reconstituted_lists['cors_default'] = [ ov['cors_default'] ]

    # OK. Next, filter out the '--internal--' stuff from our overview, and sort
    # _referenced_by.
    filtered = filtered_overview(ov)

    pretty_filtered_overview = prettify(filtered)
    pretty_reconstituted_lists = prettify(reconstituted_lists)

    udiff = list(difflib.unified_diff(pretty_filtered_overview.split("\n"),
                                      pretty_reconstituted_lists.split("\n"),
                                      fromfile="from overview", tofile="from reconstituted",
                                      lineterm=""))

    if udiff:
        errors.append("%s\n-- DIFF --\n%s\n" %
                      ("mismatch between overview and reconstituted diagnostics",
                       "\n".join(udiff)))

    return {
        '_errors': errors,
        'warnings': warnings,
        'overview': pretty_filtered_overview,
        'reconstituted': pretty_reconstituted_lists
    }

if __name__ == "__main__":
    results = diag_paranoia(sys.argv[1], ".")

    open("ov", "w").write(results['overview'])
    open("rl", "w").write(results['reconstituted'])

    if (results['warnings']):
        print("\n".join(['WARNING: %s' % x for x in results['warnings']]))

    if (results['_errors']):
        print("\n".join(['ERROR: %s' % x for x in results['_errors']]))
        sys.exit(1)
    else:
        sys.exit(0)
