import os
import subprocess

def fix_pdf():
    print("Regenerating clean HTML and PDF without header/footer noise...")
    
    current_dir = os.path.dirname(os.path.abspath(__file__))
    html_file = os.path.join(current_dir, "final_report.html")
    pdf_file = os.path.join(current_dir, "Final_Report_Document.pdf")

    # We read current html and upgrade CSS for perfect 1:1 print reproduction
    with open(html_file, 'r', encoding='utf-8') as f:
        html = f.read()

    # Update print CSS styles inside html
    new_print_css = """
        @page {
            size: A4 portrait;
            margin: 0;
        }

        @media print {
            *, *:before, *:after {
                -webkit-print-color-adjust: exact !important;
                print-color-adjust: exact !important;
            }
            html, body {
                width: 210mm;
                height: 297mm;
                margin: 0 !important;
                padding: 0 !important;
                background: #ffffff !important;
            }
            .document-wrapper {
                margin: 0 !important;
                padding: 0 !important;
                max-width: none !important;
                width: 210mm !important;
            }
            .page {
                width: 210mm !important;
                height: 297mm !important;
                margin: 0 !important;
                padding: 20mm 18mm !important;
                box-shadow: none !important;
                page-break-after: always !important;
                page-break-inside: avoid !important;
                overflow: hidden !important;
                display: flex !important;
                flex-direction: column !important;
                justify-content: space-between !important;
                box-sizing: border-box !important;
            }
        }
    """

    # Inject print-color-adjust into root style if not already present
    if "print-color-adjust: exact" not in html:
        html = html.replace("/* Cover Page */", new_print_css + "\n        /* Cover Page */")
        with open(html_file, 'w', encoding='utf-8') as f:
            f.write(html)
        print("Updated final_report.html print CSS.")

    # Execute Edge command with --no-pdf-header-footer and --include-background-graphics
    edge = r"C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe"
    if not os.path.exists(edge):
        edge = r"C:\Program Files\Google\Chrome\Application\chrome.exe"

    cmd = [
        edge,
        "--headless=new",
        "--disable-gpu",
        "--no-pdf-header-footer",
        "--include-background-graphics",
        f"--print-to-pdf={pdf_file}",
        html_file
    ]

    print("Running Edge PDF converter command...")
    res = subprocess.run(cmd, capture_output=True, text=True)
    print("Return code:", res.returncode)
    if os.path.exists(pdf_file):
        print(f"Clean PDF successfully generated! File size: {os.path.getsize(pdf_file)} bytes")

if __name__ == '__main__':
    fix_pdf()
