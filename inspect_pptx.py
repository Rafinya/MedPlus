import os
import zipfile
import xml.etree.ElementTree as ET

def inspect_pptx(pptx_path):
    print(f"\n==================== {pptx_path} ====================")
    if not os.path.exists(pptx_path):
        print("File not found.")
        return
    try:
        with zipfile.ZipFile(pptx_path, 'r') as z:
            slides = [f for f in z.namelist() if f.startswith('ppt/slides/slide') and f.endswith('.xml')]
            # sort slide names by number
            slides.sort(key=lambda x: int(x.replace('ppt/slides/slide', '').replace('.xml', '')))
            print(f"Total slides: {len(slides)}")
            for slide_name in slides:
                xml_content = z.read(slide_name)
                tree = ET.fromstring(xml_content)
                text_runs = [elem.text for elem in tree.iter() if elem.tag.endswith('}t') and elem.text]
                clean_text = " | ".join(text_runs[:10])
                print(f"{slide_name}: {clean_text[:120]}")
    except Exception as e:
        print(f"Error inspecting pptx: {e}")

if __name__ == '__main__':
    project_dir = os.path.dirname(os.path.abspath(__file__))
    inspect_pptx(os.path.join(project_dir, "Final Presentation_Template.pptx"))
    inspect_pptx(os.path.join(project_dir, "Pharmacy POS Management System (1).pptx"))
