import re

with open("internal/render/render.go", "r") as f:
    content = f.read()

content = re.sub(
    r'\t//DEBUG\n\t//for _, t := range clonedTmpl\.Templates\(\) {\n\t//\tfmt\.Printf\("Template: %s\\n", t\.Name\(\)\)\n\t//}\n',
    r'',
    content
)

with open("internal/render/render.go", "w") as f:
    f.write(content)
