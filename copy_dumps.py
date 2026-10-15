import os

def dump_file(fname, out_name):
    current_dir = os.path.dirname(os.path.abspath(__file__))
    filepath = os.path.join(current_dir, fname)
    if os.path.exists(filepath):
        with open(filepath, 'r', encoding='utf-8') as f:
            content = f.read()
        print(f"File {fname}: {len(content)} chars")
        with open(os.path.join(current_dir, out_name), 'w', encoding='utf-8') as out:
            out.write(content)

dump_file("srs_extracted.txt", "srs_clean.txt")
dump_file("test_plan_extracted.txt", "test_clean.txt")
dump_file("implementation_plan_extracted.txt", "imp_clean.txt")
