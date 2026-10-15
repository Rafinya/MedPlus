import os
import re
import zipfile
import xml.etree.ElementTree as ET
import pypdf

def extract_pdf(pdf_path):
    print(f"=== READING PDF: {pdf_path} ===")
    try:
        reader = pypdf.PdfReader(pdf_path)
        text = ""
        for i, page in enumerate(reader.pages):
            text += f"\n--- Page {i+1} ---\n" + (page.extract_text() or "")
        return text
    except Exception as e:
        return f"Error reading PDF: {e}"

def extract_doc_binary(doc_path):
    print(f"=== READING DOC (BINARY): {doc_path} ===")
    try:
        with open(doc_path, 'rb') as f:
            content = f.read()
        # Extract printable ASCII / UTF-16 text chunks
        # UTF-16LE text in .doc files often looks like alternating printable chars & null bytes
        # Let's decode utf-16-le strings and ascii strings
        raw_text = content.decode('latin1', errors='ignore')
        # match sequences of readable characters
        clean = re.sub(r'[^\x20-\x7E\r\n\t]+', ' ', raw_text)
        # also try utf-16-le
        utf16_text = content.decode('utf-16le', errors='ignore')
        clean_16 = re.sub(r'[^\x20-\x7E\r\n\t]+', ' ', utf16_text)
        return "--- LATIN1 ---\n" + clean[:5000] + "\n--- UTF16 ---\n" + clean_16[:5000]
    except Exception as e:
        return f"Error reading DOC: {e}"

def extract_pptx(pptx_path):
    print(f"=== READING PPTX: {pptx_path} ===")
    try:
        text_runs = []
        with zipfile.ZipFile(pptx_path, 'r') as z:
            for filename in z.namelist():
                if filename.startswith('ppt/slides/slide') and filename.endswith('.xml'):
                    xml_content = z.read(filename)
                    tree = ET.fromstring(xml_content)
                    for elem in tree.iter():
                        if elem.tag.endswith('}t') and elem.text:
                            text_runs.append(elem.text)
        return "\n".join(text_runs)
    except Exception as e:
        return f"Error reading PPTX: {e}"

def extract_docx(docx_path):
    print(f"=== READING DOCX: {docx_path} ===")
    try:
        import docx
        doc = docx.Document(docx_path)
        return "\n".join([p.text for p in doc.paragraphs if p.text])
    except Exception as e:
        # try zipfile extraction
        try:
            with zipfile.ZipFile(docx_path, 'r') as z:
                xml_content = z.read('word/document.xml')
                tree = ET.fromstring(xml_content)
                text_runs = [elem.text for elem in tree.iter() if elem.tag.endswith('}t') and elem.text]
                return "\n".join(text_runs)
        except Exception as e2:
            return f"Error reading DOCX: {e} / {e2}"

if __name__ == '__main__':
    project_dir = os.path.dirname(os.path.abspath(__file__))
    
    # Extract Guide doc
    guide_doc = os.path.join(project_dir, "Final Report Document-Guide.doc")
    guide_text = extract_doc_binary(guide_doc)
    with open(os.path.join(project_dir, "guide_extracted_raw.txt"), "w", encoding="utf-8") as f:
        f.write(guide_text)
    print("Guide doc extracted.")

    # Extract SRS
    srs_pdf = os.path.join(project_dir, "SRS Document.pdf")
    if os.path.exists(srs_pdf):
        srs_text = extract_pdf(srs_pdf)
        with open(os.path.join(project_dir, "srs_extracted.txt"), "w", encoding="utf-8") as f:
            f.write(srs_text)
        print("SRS extracted.")

    # Extract Implementation Plan
    imp_pdf = os.path.join(project_dir, "Implementation_Plan.pdf")
    if os.path.exists(imp_pdf):
        imp_text = extract_pdf(imp_pdf)
        with open(os.path.join(project_dir, "implementation_plan_extracted.txt"), "w", encoding="utf-8") as f:
            f.write(imp_text)
        print("Implementation Plan extracted.")

    # Extract MedPlus Test Plan
    test_pdf = os.path.join(project_dir, "MedPlus Test Plan Document.pdf")
    if os.path.exists(test_pdf):
        test_text = extract_pdf(test_pdf)
        with open(os.path.join(project_dir, "test_plan_extracted.txt"), "w", encoding="utf-8") as f:
            f.write(test_text)
        print("Test Plan extracted.")

    # Extract User Training Manual
    user_pdf = os.path.join(project_dir, "User_Training_Manual.pdf")
    if os.path.exists(user_pdf):
        user_text = extract_pdf(user_pdf)
        with open(os.path.join(project_dir, "user_manual_extracted.txt"), "w", encoding="utf-8") as f:
            f.write(user_text)
        print("User Manual extracted.")

    # Extract POS System Proposal
    prop_pdf = os.path.join(project_dir, "POS System Proposal.pdf")
    if os.path.exists(prop_pdf):
        prop_text = extract_pdf(prop_pdf)
        with open(os.path.join(project_dir, "proposal_extracted.txt"), "w", encoding="utf-8") as f:
            f.write(prop_text)
        print("Proposal extracted.")
