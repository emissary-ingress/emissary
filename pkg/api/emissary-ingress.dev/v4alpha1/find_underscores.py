import re
import sys
import os

Specials = {
    "http": "HTTP",
    "http11": "HTTP11",
    "grpc": "GRPC",
    "id": "ID",
    "url": "URL",
    "ms": "MS",
    "tls": "TLS",
    "ipv4": "IPv4",
    "ipv6": "IPv6",
    "dns": "DNS",
    "ttl": "TTL",
}

def main():
    if len(sys.argv) < 2:
        print("Please provide file paths as command line arguments")
        return

    structs = []

    for file_path in sys.argv[1:]:
        struct_info = None

        if not os.path.isfile(file_path):
            print(f"Invalid file path: {file_path}")
            continue

        file = open(file_path, "r")
        output = open(f"{os.path.basename(file_path)}", "w")

        for line in file:
            if line.strip() == "package v3alpha1":
                output.write("package v4alpha1\n")
                continue

            m = re.match(r'^\s*type\s+(\w+)\s+struct\s*{', line)

            if m:
                struct_name = m.group(1)

                struct_info = {
                    "name": struct_name,
                    "filename": file_path,
                    "fields": []
                }

                structs.append(struct_info)

            if struct_info is not None:
                m = re.search(r'`json:"([^"]*)"`', line)

                # print(f"{m is not None} - {line}")

                if m:
                    field_annotation = m.group(1)
                    all_fields = field_annotation.split(',')
                    field_name = all_fields.pop(0)
                    extras = all_fields[0] if all_fields else None

                    if '_' in field_name:
                        words = field_name.split('_')
                        if len(words) > 1:
                            camelCase = [ words.pop(0) ]

                            for word in words:
                                if word in Specials:
                                    camelCase.append(Specials[word])
                                else:
                                    camelCase.append(word.capitalize())

                        if extras:
                            camelCase.append(",")
                            camelCase.append(extras)

                        new_annotation = ''.join(camelCase)

                        struct_info["fields"].append(f"{field_annotation} -> {new_annotation}")

                        new_tags = f'json:"{new_annotation}" v3:"{field_annotation}"'
                        line = re.sub(r'`json:"[^"]*"`', f'`{new_tags}`', line)

            output.write(line)

    for struct in structs:
        if struct['fields']:
            print(f"Struct: {struct['name']} ({struct['filename']})")
            for field in struct['fields']:
                print(f"  Field: {field}")

if __name__ == "__main__":
    main()