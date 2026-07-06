import os
import glob
import re

for f in glob.glob('content/**/*.md', recursive=True):
    with open(f, 'r', encoding='utf-8') as file:
        content = file.read()

    match = re.match(r'^---\n(.*?)\n---', content, re.DOTALL)
    if match:
        fm = match.group(1)
        body = content[match.end():]

        new_lines = []
        changed = False
        for line in fm.split('\n'):
            if ':' in line and not line.startswith(' '):
                key, val = line.split(':', 1)

                if key != key.lower():
                    key = key.lower()
                    changed = True

                if key == 'author':
                    clean_val = val.strip().strip('"').strip("'")
                    if clean_val.lower() == 'jules':
                        new_val = '"Jules"'
                    elif clean_val.lower() == 'the human':
                        new_val = '"The Human"'
                    else:
                        new_val = val.strip()

                    if val.strip() != new_val:
                        val = ' ' + new_val
                        changed = True

                new_lines.append(f"{key}:{val}")
            else:
                new_lines.append(line)

        if changed:
            new_fm = '\n'.join(new_lines)
            new_content = f"---\n{new_fm}\n---{body}"
            with open(f, 'w', encoding='utf-8') as file:
                file.write(new_content)
            print(f"Updated {f}")
