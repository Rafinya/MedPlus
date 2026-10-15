import os
import re

current_dir = os.path.dirname(os.path.abspath(__file__))
doc_path = os.path.join(current_dir, "Final Report Document-Guide.doc")
if not os.path.exists(doc_path):
    # fallback to available doc
    for candidate in ["User manual guide.doc", "Implementation Plan -Guide.docx"]:
        cand_path = os.path.join(current_dir, candidate)
        if os.path.exists(cand_path):
            doc_path = cand_path
            break

with open(doc_path, "rb") as f:
    data = f.read()

# Extract UTF-16LE text sequences
text16 = data.decode("utf-16le", errors="ignore")
lines16 = [line.strip() for line in re.split(r'[\r\n\x00]+', text16) if len(line.strip()) > 3]

# Extract ASCII / Latin-1 text sequences
text8 = data.decode("latin1", errors="ignore")
lines8 = [line.strip() for line in re.split(r'[\r\n]+', text8) if len(line.strip()) > 3]

print("=== UTF16 STRINGS (sample) ===")
for l in lines16[:40]:
    # filter out binary noise
    clean = "".join([c for c in l if 32 <= ord(c) <= 126 or ord(c) in (9, 10, 13)])
    if len(clean) > 3:
        print(clean)

print("\n=== LATIN1 STRINGS (sample) ===")
for l in lines8[:40]:
    clean = "".join([c for c in l if 32 <= ord(c) <= 126 or ord(c) in (9, 10, 13)])
    if len(clean) > 3:
        print(clean)
