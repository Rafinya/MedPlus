import os

def read_summary(filename):
    print(f"\n==================== {filename} ====================")
    current_dir = os.path.dirname(os.path.abspath(__file__))
    filepath = os.path.join(current_dir, filename)
    if not os.path.exists(filepath):
        print("File not found.")
        return
    with open(filepath, 'r', encoding='utf-8') as f:
        text = f.read()
    lines = text.splitlines()
    print(f"Total lines: {len(lines)}")
    # Print section headers or key lines
    for line in lines:
        if line.strip().startswith("--- Page") or line.strip().startswith("1.") or line.strip().startswith("2.") or line.strip().startswith("3.") or line.strip().startswith("4.") or line.strip().startswith("5.") or line.strip().startswith("Chapter") or "SECTION" in line.upper() or "TABLE OF CONTENTS" in line.upper():
            print(line[:100])

read_summary("srs_extracted.txt")
read_summary("implementation_plan_extracted.txt")
read_summary("test_plan_extracted.txt")
read_summary("user_manual_extracted.txt")
read_summary("proposal_extracted.txt")
