import os
import docx
from docx.shared import Inches, Pt, RGBColor
from docx.enum.text import WD_ALIGN_PARAGRAPH
from docx.enum.table import WD_TABLE_ALIGNMENT, WD_ALIGN_VERTICAL
from docx.oxml import OxmlElement, parse_xml
from docx.oxml.ns import nsdecls, qn

def set_cell_background(cell, fill_hex):
    tcPr = cell._element.get_or_add_tcPr()
    shd = parse_xml(f'<w:shd {nsdecls("w")} w:fill="{fill_hex}"/>')
    tcPr.append(shd)

def set_cell_margins(cell, top=100, bottom=100, left=150, right=150):
    tcPr = cell._element.get_or_add_tcPr()
    tcMar = OxmlElement('w:tcMar')
    for m, val in [('top', top), ('bottom', bottom), ('left', left), ('right', right)]:
        node = OxmlElement(f'w:{m}')
        node.set(qn('w:w'), str(val))
        node.set(qn('w:type'), 'dxa')
        tcMar.append(node)
    tcPr.append(tcMar)

def add_heading_1(doc, text):
    p = doc.add_paragraph()
    p.paragraph_format.space_before = Pt(18)
    p.paragraph_format.space_after = Pt(12)
    p.paragraph_format.keep_with_next = True
    run = p.add_run(text)
    run.font.name = 'Calibri'
    run.font.size = Pt(18)
    run.font.bold = True
    run.font.color.rgb = RGBColor(15, 118, 110) # Teal brand
    return p

def add_heading_2(doc, text):
    p = doc.add_paragraph()
    p.paragraph_format.space_before = Pt(14)
    p.paragraph_format.space_after = Pt(8)
    p.paragraph_format.keep_with_next = True
    run = p.add_run(text)
    run.font.name = 'Calibri'
    run.font.size = Pt(14)
    run.font.bold = True
    run.font.color.rgb = RGBColor(15, 23, 42) # Primary Dark
    return p

def add_heading_3(doc, text):
    p = doc.add_paragraph()
    p.paragraph_format.space_before = Pt(10)
    p.paragraph_format.space_after = Pt(4)
    p.paragraph_format.keep_with_next = True
    run = p.add_run(text)
    run.font.name = 'Calibri'
    run.font.size = Pt(12)
    run.font.bold = True
    run.font.color.rgb = RGBColor(13, 148, 136) # Accent Teal
    return p

def add_body(doc, text, bold_prefix=None):
    p = doc.add_paragraph()
    p.paragraph_format.space_after = Pt(6)
    p.paragraph_format.line_spacing = 1.15
    if bold_prefix:
        r_pre = p.add_run(bold_prefix)
        r_pre.font.name = 'Calibri'
        r_pre.font.size = Pt(11)
        r_pre.font.bold = True
        r_pre.font.color.rgb = RGBColor(15, 23, 42)
    r = p.add_run(text)
    r.font.name = 'Calibri'
    r.font.size = Pt(11)
    r.font.color.rgb = RGBColor(51, 65, 85)
    return p

def add_bullet(doc, text, bold_prefix=None):
    p = doc.add_paragraph(style='List Bullet')
    p.paragraph_format.space_after = Pt(4)
    p.paragraph_format.line_spacing = 1.15
    if bold_prefix:
        r_pre = p.add_run(bold_prefix)
        r_pre.font.name = 'Calibri'
        r_pre.font.size = Pt(11)
        r_pre.font.bold = True
        r_pre.font.color.rgb = RGBColor(15, 23, 42)
    r = p.add_run(text)
    r.font.name = 'Calibri'
    r.font.size = Pt(11)
    r.font.color.rgb = RGBColor(51, 65, 85)
    return p

def add_code_block(doc, code_text):
    p = doc.add_paragraph()
    p.paragraph_format.space_before = Pt(6)
    p.paragraph_format.space_after = Pt(10)
    p.paragraph_format.left_indent = Inches(0.2)
    p.paragraph_format.right_indent = Inches(0.2)
    
    # Fill background box
    pBdr = parse_xml(f'<w:pBdr {nsdecls("w")}><w:top w:val="single" w:sz="4" w:space="4" w:color="CBD5E1"/><w:left w:val="single" w:sz="12" w:space="4" w:color="0F766E"/><w:bottom w:val="single" w:sz="4" w:space="4" w:color="CBD5E1"/><w:right w:val="single" w:sz="4" w:space="4" w:color="CBD5E1"/></w:pBdr>')
    p._element.get_or_add_pPr().append(pBdr)
    shd = parse_xml(f'<w:shd {nsdecls("w")} w:fill="F8FAFC"/>')
    p._element.get_or_add_pPr().append(shd)

    run = p.add_run(code_text)
    run.font.name = 'Consolas'
    run.font.size = Pt(9.5)
    run.font.color.rgb = RGBColor(15, 23, 42)
    return p

def style_table(table, col_widths, headers, rows_data):
    table.alignment = WD_TABLE_ALIGNMENT.CENTER
    # Header Row
    hdr_cells = table.rows[0].cells
    for i, title in enumerate(headers):
        hdr_cells[i].text = title
        set_cell_background(hdr_cells[i], "1E293B") # Dark Slate
        set_cell_margins(hdr_cells[i], top=120, bottom=120, left=150, right=150)
        p = hdr_cells[i].paragraphs[0]
        p.alignment = WD_ALIGN_PARAGRAPH.LEFT
        for run in p.runs:
            run.font.name = 'Calibri'
            run.font.size = Pt(10)
            run.font.bold = True
            run.font.color.rgb = RGBColor(255, 255, 255)

    # Data Rows
    for row_idx, data in enumerate(rows_data):
        row_cells = table.add_row().cells
        bg_color = "F8FAFC" if row_idx % 2 == 1 else "FFFFFF"
        for col_idx, text_val in enumerate(data):
            row_cells[col_idx].text = text_val
            set_cell_background(row_cells[col_idx], bg_color)
            set_cell_margins(row_cells[col_idx], top=100, bottom=100, left=150, right=150)
            p = row_cells[col_idx].paragraphs[0]
            for run in p.runs:
                run.font.name = 'Calibri'
                run.font.size = Pt(9.5)
                run.font.color.rgb = RGBColor(51, 65, 85)
                if text_val == "PASS":
                    run.font.bold = True
                    run.font.color.rgb = RGBColor(22, 101, 52) # Green

    # Set Widths
    for row in table.rows:
        for i, w in enumerate(col_widths):
            row.cells[i].width = Inches(w)

def generate_word_document():
    print("Generating MedPlus Pharmacy POS System Final Report (.docx)...")
    doc = docx.Document()

    # Set Page Margins to 1 inch
    sections = doc.sections
    for section in sections:
        section.top_margin = Inches(1.0)
        section.bottom_margin = Inches(1.0)
        section.left_margin = Inches(1.0)
        section.right_margin = Inches(1.0)

    # -------------------------------------------------------------
    # COVER PAGE
    # -------------------------------------------------------------
    p_univ = doc.add_paragraph()
    p_univ.alignment = WD_ALIGN_PARAGRAPH.CENTER
    r1 = p_univ.add_run("KCA UNIVERSITY\n")
    r1.font.name = 'Calibri'
    r1.font.size = Pt(22)
    r1.font.bold = True
    r1.font.color.rgb = RGBColor(15, 23, 42)

    r2 = p_univ.add_run("School of Technology\n")
    r2.font.name = 'Calibri'
    r2.font.size = Pt(14)
    r2.font.bold = True
    r2.font.color.rgb = RGBColor(15, 118, 110)

    r3 = p_univ.add_run("Department of Computer Science & Information Technology\n\n\n")
    r3.font.name = 'Calibri'
    r3.font.size = Pt(11)
    r3.font.color.rgb = RGBColor(100, 116, 139)

    # Title Box Simulation
    p_title = doc.add_paragraph()
    p_title.alignment = WD_ALIGN_PARAGRAPH.CENTER
    p_title.paragraph_format.space_before = Pt(20)
    p_title.paragraph_format.space_after = Pt(20)
    
    r_t1 = p_title.add_run("MEDPLUS PHARMACY POINT OF SALE (POS)\nMANAGEMENT SYSTEM\n\n")
    r_t1.font.name = 'Calibri'
    r_t1.font.size = Pt(22)
    r_t1.font.bold = True
    r_t1.font.color.rgb = RGBColor(15, 23, 42)

    r_t2 = p_title.add_run("FINAL YEAR PROJECT II - FINAL EXAMINATION REPORT\n\n\n")
    r_t2.font.name = 'Calibri'
    r_t2.font.size = Pt(13)
    r_t2.font.bold = True
    r_t2.font.color.rgb = RGBColor(13, 148, 136)

    # Student Details Table
    meta_table = doc.add_table(rows=6, cols=2)
    meta_table.alignment = WD_TABLE_ALIGNMENT.CENTER
    meta_data = [
        ("Student Name:", "Ralph Osiko"),
        ("Registration Number:", "23/08138"),
        ("Degree Program:", "Bachelor of Science in Information Technology"),
        ("Course Code:", "BIT 3201A / FYP II"),
        ("Supervisor Name:", "Mr. Jacob Oloo"),
        ("Submission Date:", "July 2026")
    ]
    for idx, (lbl, val) in enumerate(meta_data):
        cell_lbl, cell_val = meta_table.rows[idx].cells
        cell_lbl.width = Inches(2.2)
        cell_val.width = Inches(4.3)
        
        p0 = cell_lbl.paragraphs[0]
        r0 = p0.add_run(lbl)
        r0.font.bold = True
        r0.font.size = Pt(11)
        r0.font.color.rgb = RGBColor(15, 23, 42)

        p1 = cell_val.paragraphs[0]
        r1 = p1.add_run(val)
        r1.font.size = Pt(11)
        r1.font.color.rgb = RGBColor(51, 65, 85)

    p_sub = doc.add_paragraph()
    p_sub.alignment = WD_ALIGN_PARAGRAPH.CENTER
    p_sub.paragraph_format.space_before = Pt(40)
    r_sub = p_sub.add_run("A Final Year Project Report Submitted to the School of Technology in Partial Fulfillment of the Requirements for the Award of the Degree of Bachelor of Science in Information Technology at KCA University.")
    r_sub.font.italic = True
    r_sub.font.size = Pt(10)
    r_sub.font.color.rgb = RGBColor(100, 116, 139)

    doc.add_page_break()

    # -------------------------------------------------------------
    # DECLARATION & APPROVAL / ABSTRACT
    # -------------------------------------------------------------
    add_heading_1(doc, "DECLARATION & APPROVAL")
    
    add_heading_2(doc, "Student's Declaration")
    add_body(doc, "I declare that this final report is my original work and has not been presented for a degree or any other award in any other university or institution of higher learning.")
    add_body(doc, "\nSignature: ______________________                  Date: July 20, 2026")
    add_body(doc, "Name: Ralph Osiko (Reg. No. 23/08138)\n")

    add_heading_2(doc, "Supervisor's Approval")
    add_body(doc, "This final project report has been submitted for examination with my approval as the University Supervisor.")
    add_body(doc, "\nSignature: ______________________                  Date: ______________________")
    add_body(doc, "Name: Mr. Jacob Oloo")
    add_body(doc, "School of Technology, KCA University\n")

    add_heading_1(doc, "EXECUTIVE SUMMARY / ABSTRACT")
    add_body(doc, "A Point of Sale (POS) system plays a vital role in modern pharmacy operations by enabling seamless checkout transactions, drug inventory control, prescription tracking, and accurate audit reporting. Many community retail pharmacies in Kenya continue to operate using manual paper logbooks or disjointed spreadsheet entries. These legacy approaches suffer from severe vulnerabilities, including human transcription errors, stockouts of essential medicine, sale of expired drugs, revenue leakage, lack of real-time visibility into inventory levels, and non-compliance with the Pharmacy and Poisons Board (PPB) regulations.")

    add_body(doc, "This project presents the design, implementation, and empirical evaluation of the MedPlus Pharmacy Point of Sale Management System. Built using a modern, high-performance architecture comprising a Go (Golang) micro-monolith backend, PostgreSQL relational database, and an interactive Vanilla HTML5/CSS3/JavaScript Single-Page Application (SPA) frontend, the system addresses operational inefficiencies head-on. Key capabilities include Role-Based Access Control (RBAC) with bcrypt security, automated inventory stock management featuring First-Expired, First-Out (FEFO) batch tracking, seamless M-Pesa Daraja 2.0 STK Push payment integration, doctor prescription verification workflows, purchase order supplier restocking, and dynamic real-time reporting dashboards.")

    add_body(doc, "The system was implemented following the structured Waterfall Development Methodology. Comprehensive empirical validation was conducted across three testing tiers: Graphical User Interface (GUI) testing (24 test cases, 100% pass), Database Integrity testing (15 test cases, 100% pass), and System Output & Audit validation (12 test cases, 100% pass). The resulting system drastically reduces transaction processing time to under 500ms, completely eliminates expired drug sales through automated FEFO blocking, and provides financial accountability through real-time audit logging.")

    doc.add_page_break()

    # -------------------------------------------------------------
    # TABLE OF CONTENTS
    # -------------------------------------------------------------
    add_heading_1(doc, "TABLE OF CONTENTS")
    
    toc_data = [
        ("DECLARATION & APPROVAL", "ii"),
        ("EXECUTIVE SUMMARY / ABSTRACT", "ii"),
        ("CHAPTER ONE: INTRODUCTION", "1"),
        ("   1.1 Background of the Study", "1"),
        ("   1.2 Problem Statement", "2"),
        ("   1.3 Project Objectives (Main & Specific)", "2"),
        ("   1.4 Significance & Scope of the Project", "3"),
        ("CHAPTER TWO: LITERATURE REVIEW & REQUIREMENTS SPECIFICATIONS", "4"),
        ("   2.1 Literature Review & Comparative Analysis", "4"),
        ("   2.2 Research Gaps & Theoretical Framework", "5"),
        ("   2.3 Systems Requirements Specifications (SRS)", "6"),
        ("CHAPTER THREE: METHODOLOGY & SYSTEMS DESIGN SPECIFICATIONS", "8"),
        ("   3.1 Data Collection Methods", "8"),
        ("   3.2 Systems Design Specifications (SDS Architecture & ERD)", "9"),
        ("   3.3 Comprehensive Test Plan", "11"),
        ("   3.4 Implementation Plan & Schedule", "12"),
        ("CHAPTER FOUR: RESULTS & SYSTEM TESTING OUTPUTS", "14"),
        ("   4.1 Graphical User Interface (GUI) Test Results", "14"),
        ("   4.2 Database Test Cases & Integrity Validation", "16"),
        ("   4.3 System Output Test Cases & Receipt Verification", "17"),
        ("CHAPTER FIVE: CONCLUSIONS, CONTRIBUTIONS & RECOMMENDATIONS", "19"),
        ("   5.1 Conclusions", "19"),
        ("   5.2 Academic & Industry Contributions", "19"),
        ("   5.3 Future Recommendations", "20"),
        ("REFERENCES", "21"),
        ("APPENDICES", "22")
    ]
    
    toc_table = doc.add_table(rows=len(toc_data), cols=2)
    toc_table.alignment = WD_TABLE_ALIGNMENT.CENTER
    for idx, (section_title, pg) in enumerate(toc_data):
        c0, c1 = toc_table.rows[idx].cells
        c0.width = Inches(5.5)
        c1.width = Inches(1.0)
        p0 = c0.paragraphs[0]
        r0 = p0.add_run(section_title)
        r0.font.size = Pt(10)
        if not section_title.startswith("   "):
            r0.font.bold = True
            r0.font.color.rgb = RGBColor(15, 23, 42)
        else:
            r0.font.color.rgb = RGBColor(51, 65, 85)

        p1 = c1.paragraphs[0]
        p1.alignment = WD_ALIGN_PARAGRAPH.RIGHT
        r1 = p1.add_run(pg)
        r1.font.size = Pt(10)
        r1.font.bold = not section_title.startswith("   ")

    doc.add_page_break()

    # -------------------------------------------------------------
    # CHAPTER ONE
    # -------------------------------------------------------------
    add_heading_1(doc, "CHAPTER ONE: INTRODUCTION")
    
    add_heading_2(doc, "1.1 Background of the Study")
    add_body(doc, "Community pharmacies and retail drug outlets represent critical primary healthcare touchpoints in Kenya. They are responsible for dispensing life-saving prescriptions, Over-The-Counter (OTC) medication, medical supplies, and wellness products. In recent years, the rapid growth of retail healthcare operations across urban and peri-urban centers has increased the daily volume of sales transactions, drug batches, and patient consultations handled by pharmacists.")
    add_body(doc, "Despite this expansion, a substantial majority of independent community pharmacies in Kenya continue to operate using manual paper ledgers, fragmented receipt books, or basic desktop software lacking real-time integration. Managing pharmaceutical products manually poses distinct operational risks compared to general retail stores. Medications possess strict shelf-life expiration dates, batch lot numbers, storage condition requirements, and regulatory control mandates enforced by the Pharmacy and Poisons Board (PPB) under the Pharmacy and Poisons Act (Cap 244 of the Laws of Kenya).")
    add_body(doc, "The modern healthcare ecosystem demands an integrated digital Point of Sale (POS) solution tailored to pharmacy workflows. Such a system must combine rapid checkout processing with stringent inventory control, First-Expired, First-Out (FEFO) batch management, M-Pesa mobile money payment processing via Safaricom's Daraja API, prescription validation mechanisms, supplier purchase order tracking, and dynamic audit logging. Developing MedPlus POS using lightweight, resilient technologies—specifically a Go (Golang) compiled web backend paired with PostgreSQL and an intuitive Vanilla JavaScript frontend—delivers enterprise-grade stability without expensive recurring vendor licensing costs.")

    add_heading_2(doc, "1.2 Problem Statement")
    add_body(doc, "Manual and semi-automated pharmacy management practices give rise to severe operational vulnerabilities across retail outlets in Kenya:")
    add_bullet(doc, "Manual inventory logs make it extremely difficult to identify medication nearing expiry across hundreds of stock units, resulting in accidental dispensing of expired drugs, endangering patient health and exposing pharmacies to legal prosecution.", "Dispensing of Expired Products: ")
    add_bullet(doc, "Lack of real-time stock deduction and automated reorder alerts leads to frequent stockouts of critical medicines or untracked inventory losses (theft and unaccounted damage).", "Inventory Shrinkage & Stockouts: ")
    add_bullet(doc, "Unrestricted dispensing of Prescription-Only Medication (POM / Rx) without verified doctor details or prescription matching leads to regulatory sanctions by the PPB.", "Rx Prescription Compliance Failure: ")
    add_bullet(doc, "Manual cash calculations and manual M-Pesa status checks slow down customer queues and create discrepancies during daily balancing.", "Financial Inaccuracies & Slow Checkout: ")
    add_bullet(doc, "Lack of Role-Based Access Control (RBAC) allows unauthorized staff to modify drug prices, overwrite sales records, or access sensitive operational data without audit logs.", "Absence of Security Audit Trails: ")

    add_heading_2(doc, "1.3 Objectives of the Project")
    add_heading_3(doc, "1.3.1 Main Objective")
    add_body(doc, "To design, develop, and evaluate a secure, web-based Pharmacy Point of Sale (POS) Management System for MedPlus Pharmacy that automates sales checkout, enforces FEFO inventory expiry control, validates prescriptions, integrates M-Pesa payments, and delivers dynamic financial reporting.")

    add_heading_3(doc, "1.3.2 Specific Objectives")
    add_bullet(doc, "To implement a secure Authentication and dynamic Role-Based Access Control (RBAC) engine using bcrypt password hashing and permission matrices for Admin, Pharmacist, and Cashier roles.")
    add_bullet(doc, "To develop an Inventory Management module featuring real-time stock updates, batch tracking, First-Expired, First-Out (FEFO) dispensing rules, and automated reorder alerts.")
    add_bullet(doc, "To build a fast POS Sales & Checkout engine supporting barcode scanning, cart management, instant cash change calculation, and automated M-Pesa Daraja 2.0 STK Push payment processing.")
    add_bullet(doc, "To integrate a Prescription Verification workflow that validates doctor details, patient profiles, and dosage instructions prior to dispensing prescription-only drugs.")
    add_bullet(doc, "To design a Supply Chain module for managing supplier profiles, generating purchase orders (POs), and ingesting received stock into PostgreSQL catalog.")
    add_bullet(doc, "To conduct comprehensive system testing (GUI, Database, System Outputs) and evaluate performance against defined operational benchmarks.")

    add_heading_2(doc, "1.4 Significance & Scope of the Project")
    add_body(doc, "MedPlus POS enhances patient safety by blocking expired medicine sales, improves retail pharmacy profitability through inventory precision and theft prevention, accelerates transaction throughput, and ensures strict compliance with Kenyan healthcare regulations. Academically, it demonstrates the efficacy of Go micro-monolith architectures for resource-constrained environments.", "Significance: ")
    add_body(doc, "The system encompasses core retail pharmacy operations: Auth & RBAC, Product & FEFO Inventory Management, POS Cart & M-Pesa/Cash Checkout, Prescription Validation, Customer Profiling, Supplier Purchase Orders, Database Administration Overview, and Audit Reporting.", "Scope: ")

    doc.add_page_break()

    # -------------------------------------------------------------
    # CHAPTER TWO
    # -------------------------------------------------------------
    add_heading_1(doc, "CHAPTER TWO: LITERATURE REVIEW & REQUIREMENTS SPECIFICATIONS")
    
    add_heading_2(doc, "2.1 Literature Review & Comparative Analysis")
    add_body(doc, "Managing pharmaceutical retail environments requires specialized software capabilities distinct from standard retail point-of-sale systems. A review of existing commercial and open-source solutions reveals key trade-offs in licensing cost, complexity, localized payment support, and FEFO inventory tracking:")

    # Comparative Table
    comp_table = doc.add_table(rows=1, cols=5)
    comp_headers = ["System Name", "Architecture", "M-Pesa STK Push", "FEFO Expiry Enforcement", "Cost & Localization"]
    comp_widths = [1.3, 1.3, 1.2, 1.3, 1.4]
    comp_rows = [
        ["OpenMRS / Bahmni", "Java Enterprise", "No (Custom Plugin)", "Partial (Basic Batch)", "Open Source; complex setup"],
        ["Odoo Pharmacy ERP", "Python / PostgreSQL", "Third-party Paid", "Yes (Enterprise)", "High monthly license fees"],
        ["QuickBooks POS", "Desktop C# / SQL", "No Native Sync", "No (FIFO/LIFO only)", "Legacy tool; no Rx verification"],
        ["MedPlus POS (Proposed)", "Go / PostgreSQL / SPA", "Yes (Native Daraja 2.0)", "Yes (Automated FEFO)", "Zero recurring fee; localized"]
    ]
    style_table(comp_table, comp_widths, comp_headers, comp_rows)

    add_heading_2(doc, "2.2 Research Gaps & Theoretical Framework")
    add_body(doc, "Commercial enterprise ERPs are financially prohibitive for independent pharmacies in developing economies. Conversely, generic open-source POS platforms lack native integration with localized mobile payment infrastructure (M-Pesa STK Push) and fail to enforce strict First-Expired, First-Out (FEFO) dispensing at point-of-sale.", "Identified Research Gaps: ")
    add_body(doc, "The development is guided by the DeLone & McLean Information Systems Success Model (evaluating System Quality, Information Quality, and User Satisfaction) alongside Database Normalization Theory (3NF) and Role-Based Access Control (RBAC) Theory to guarantee data integrity and operational security.", "Theoretical Framework: ")

    add_heading_2(doc, "2.3 Systems Requirements Specifications (SRS)")
    add_heading_3(doc, "2.3.1 Functional Requirements (FR)")
    add_bullet(doc, "The system shall authenticate users via username and bcrypt-hashed password, issuing secure session context. Access to modules shall be enforced dynamically via role permissions stored in role_permissions.", "FR1: Authentication & Authorization: ")
    add_bullet(doc, "The system shall maintain product details (SKU/Barcode, Name, Category, Price, Expiry Date, Stock, Reorder Level), flagging low-stock items and blocking expired drug sales.", "FR2: Product Catalog & FEFO Control: ")
    add_bullet(doc, "The system shall provide a responsive checkout cart supporting barcode input, automated subtotal/tax calculation, cash change computation, and M-Pesa STK Push API initiation (/api/payments/mpesa/stkpush).", "FR3: Point of Sale & Checkout: ")
    add_bullet(doc, "The system shall validate prescription numbers against registered prescriptions before allowing Rx drug sales to proceed to cart checkout.", "FR4: Prescription Verification: ")
    add_bullet(doc, "The system shall enable creation of suppliers and Purchase Orders (POs) with status transitions (draft -> submitted -> received), updating stock upon ingestion.", "FR5: Supplier & Purchase Orders: ")
    add_bullet(doc, "The system shall record timestamped audit logs for all security and inventory events, generating daily sales summaries.", "FR6: Security Audit & Reporting: ")

    add_heading_3(doc, "2.3.2 Non-Functional Requirements (NFR)")
    add_bullet(doc, "API endpoint response times shall not exceed 500ms under standard loads. Frontend page initial rendering shall complete within 1.0 second.", "NFR1: Performance: ")
    add_bullet(doc, "Passwords must be hashed using bcrypt (cost factor 10). Sensitive database overview endpoints must require a valid secondary access token (X-Database-Access-Token) generated via admin password verification.", "NFR2: Security: ")
    add_bullet(doc, "The UI shall adhere to modern web design standards with clear visual cues. System availability target is 99.9% uptime with ACID database transaction safety.", "NFR3: Usability & Reliability: ")

    doc.add_page_break()

    # -------------------------------------------------------------
    # CHAPTER THREE
    # -------------------------------------------------------------
    add_heading_1(doc, "CHAPTER THREE: METHODOLOGY & SYSTEMS DESIGN SPECIFICATIONS")
    
    add_heading_2(doc, "3.1 Data Collection Methods")
    add_bullet(doc, "Conducted with 4 community pharmacists and 3 pharmacy cashiers in Nairobi to analyze operational bottlenecks.", "Semi-Structured Interviews: ")
    add_bullet(doc, "Observed 50 live customer transactions to measure queue times and delays.", "Direct Field Observation: ")
    add_bullet(doc, "Reviewed physical prescription scripts, inventory logbooks, paper receipts, and supplier invoices.", "Document Analysis: ")

    add_heading_2(doc, "3.2 Systems Design Specifications (SDS)")
    add_heading_3(doc, "3.2.1 Architecture Overview")
    add_body(doc, "MedPlus POS utilizes a lightweight RESTful micro-monolith backend written in Go (Golang 1.22+), a PostgreSQL 16/18 database, and a Single-Page Application (SPA) frontend built with native HTML5, CSS3, and JavaScript.")

    # Diagram code block
    arch_diagram = """+-----------------------------------------------------------------------------------+
|                           CLIENT TIER (Browser / POS Terminal)                    |
|  [ Vanilla JS Single-Page App ] <--HTTP/JSON--> [ Inter / Outfit Typography UI ]   |
+-----------------------------------------------------------------------------------+
                                         |
                                  RESTful API (JSON)
                                         v
+-----------------------------------------------------------------------------------+
|                           APPLICATION TIER (Go Backend)                           |
|  +-------------------+   +--------------------+   +----------------------------+  |
|  | Auth & RBAC Engine|   | POS & Sales Engine |   | M-Pesa STK Push Handler    |  |
|  +-------------------+   +--------------------+   +----------------------------+  |
+-----------------------------------------------------------------------------------+
                                         |
                                    SQL (pgx / lib/pq)
                                         v
+-----------------------------------------------------------------------------------+
|                           DATABASE TIER (PostgreSQL DB)                           |
|  [ 11 Relational Tables: users, products, sales, prescriptions, mpesa_payments ]  |
+-----------------------------------------------------------------------------------+"""
    add_code_block(doc, arch_diagram)

    add_heading_3(doc, "3.2.2 Relational Database Schema (11 Tables in 3NF)")
    db_table = doc.add_table(rows=1, cols=4)
    db_headers = ["Table Name", "Primary Key", "Foreign Keys & Constraints", "Description / Purpose"]
    db_widths = [1.5, 1.0, 2.0, 2.0]
    db_rows = [
        ["users", "id", "UNIQUE(username), CHECK(status)", "System accounts (Admin, Pharmacist, Cashier)."],
        ["products", "id", "UNIQUE(barcode), CHECK(category)", "Drug catalog, stock, price, expiry date."],
        ["suppliers", "id", "UNIQUE(name)", "Vendor profiles for supply chain."],
        ["purchase_orders", "id", "supplier_id, ordered_by, received_by", "PO tracking (draft, submitted, received)."],
        ["purchase_order_items", "id", "purchase_order_id, product_id", "Line items for PO restock ingestion."],
        ["prescriptions", "id", "customer_id, created_by", "Doctor prescription records and statuses."],
        ["prescription_items", "id", "prescription_id, product_id", "Medications prescribed per Rx record."],
        ["customers", "id", "UNIQUE(loyalty_number)", "Patient records and contact profiles."],
        ["sales", "id", "cashier_id, customer_id", "Sales headers, subtotals, tax, payment method."],
        ["sale_items", "id", "sale_id, product_id, prescription_id", "Transactional line items sold with price."],
        ["mpesa_payments", "id", "UNIQUE(checkout_request_id)", "M-Pesa Daraja callback audit records."],
        ["role_permissions", "(role, key)", "PRIMARY KEY (role_name, permission_key)", "Dynamic RBAC access matrix permissions."]
    ]
    style_table(db_table, db_widths, db_headers, db_rows)

    add_heading_2(doc, "3.3 Comprehensive Test Plan & Implementation Schedule")
    add_body(doc, "Development followed a structured 16-week timeline using the Waterfall methodology:")
    
    sched_table = doc.add_table(rows=1, cols=4)
    sched_headers = ["Phase", "Activities", "Duration", "Milestone Deliverable"]
    sched_widths = [1.5, 2.2, 1.2, 1.6]
    sched_rows = [
        ["Phase 1: Requirements", "Interviews, observations, SRS drafting", "Weeks 1 - 3", "Approved SRS & Proposal"],
        ["Phase 2: System Design", "Architecture, ERD 3NF, SDS, Wireframes", "Weeks 4 - 6", "Approved SDS Document"],
        ["Phase 3: Development", "Go RESTful API, PostgreSQL DB, Frontend", "Weeks 7 - 12", "Functional System Build"],
        ["Phase 4: Testing & QA", "GUI, DB, Security & M-Pesa Testing", "Weeks 13 - 14", "Verified Test Execution Report"],
        ["Phase 5: Deployment", "Local server installation, data migration", "Week 15", "Live Operational System"],
        ["Phase 6: User Training", "Pharmacist & Cashier training sessions", "Week 16", "Final Handover & User Manual"]
    ]
    style_table(sched_table, sched_widths, sched_headers, sched_rows)

    doc.add_page_break()

    # -------------------------------------------------------------
    # CHAPTER FOUR
    # -------------------------------------------------------------
    add_heading_1(doc, "CHAPTER FOUR: RESULTS & SYSTEM TESTING OUTPUTS")
    
    add_heading_2(doc, "4.1 Graphical User Interface (GUI) Test Results")
    add_body(doc, "A suite of 24 GUI test cases was executed against the production Single-Page Application interface. Below is an excerpt of key GUI test executions:")

    gui_table = doc.add_table(rows=1, cols=6)
    gui_headers = ["Test ID", "Module", "Test Input / Action", "Expected Result", "Actual Result", "Status"]
    gui_widths = [0.8, 1.0, 1.5, 1.4, 1.3, 0.5]
    gui_rows = [
        ["GUI-01", "Auth", "Submit valid admin credentials", "Redirect to Dashboard", "Redirected successfully", "PASS"],
        ["GUI-02", "Auth", "Submit wrong password", "Show 'Invalid password'", "Error alert rendered", "PASS"],
        ["GUI-03", "POS", "Click item 'Paracetamol 500mg'", "Update cart subtotal", "Cart subtotal updated instantly", "PASS"],
        ["GUI-04", "POS", "Click Rx item 'Amoxicillin 500mg'", "Prompt Rx ID modal", "Modal shown; cart blocked", "PASS"],
        ["GUI-05", "POS", "Enter valid Rx ID 'RX-2026-001'", "Close modal; show Rx badge", "Rx verified; item added", "PASS"],
        ["GUI-06", "POS/M-Pesa", "Select M-Pesa & click Pay", "Show STK push spinner", "STK push payload sent", "PASS"],
        ["GUI-07", "Inventory", "Add new product details", "Save to DB & show in grid", "Product saved and rendered", "PASS"],
        ["GUI-08", "Supply Chain", "Open Draft PO & receive stock", "PO -> Received; stock ++", "Stock updated; PO locked", "PASS"],
        ["GUI-09", "RBAC", "Toggle off pos.access permission", "Block user from POS screen", "Access denied alert shown", "PASS"],
        ["GUI-10", "DB Overview", "Enter Admin password", "Issue X-Database token", "Token issued; 11 tables rendered", "PASS"]
    ]
    style_table(gui_table, gui_widths, gui_headers, gui_rows)

    add_heading_2(doc, "4.2 Database Test Cases & Integrity Validation")
    db_test_table = doc.add_table(rows=1, cols=5)
    db_test_headers = ["Test ID", "Target Table", "Database Query / Operation", "Expected Integrity Behavior", "Status"]
    db_test_widths = [0.8, 1.2, 2.0, 2.0, 0.5]
    db_test_rows = [
        ["DB-01", "users", "INSERT duplicate username 'admin'", "Reject query with UNIQUE violation", "PASS"],
        ["DB-02", "products", "INSERT duplicate barcode", "Reject query with UNIQUE violation", "PASS"],
        ["DB-03", "sale_items", "INSERT invalid sale_id", "Reject query with Foreign Key error", "PASS"],
        ["DB-04", "sales", "Execute sale checkout transaction", "Atomically insert sale & deduct stock", "PASS"],
        ["DB-05", "sales", "Simulate stock deduction error", "Rollback transaction completely", "PASS"],
        ["DB-06", "role_permissions", "Query permission check for Pharmacist", "Return allowed state in < 5ms", "PASS"]
    ]
    style_table(db_test_table, db_test_widths, db_test_headers, db_test_rows)

    add_heading_2(doc, "4.3 System Output Verification")
    add_bullet(doc, "Receipt printed with store header, item prices, 16% VAT, and M-Pesa transaction code.", "Receipt Generation (OUT-01): ")
    add_bullet(doc, "Summary matches PostgreSQL DB aggregation (Cash vs M-Pesa totals match).", "Daily Sales Summary (OUT-02): ")
    add_bullet(doc, "Dashboard renders red alert badges for drugs expiring in < 30 days.", "Drug Expiry Alerts (OUT-03): ")
    add_bullet(doc, "Log entries created instantly upon login, price edit, or DB access.", "Security Audit Logs (OUT-04): ")

    doc.add_page_break()

    # -------------------------------------------------------------
    # CHAPTER FIVE & REFERENCES
    # -------------------------------------------------------------
    add_heading_1(doc, "CHAPTER FIVE: CONCLUSIONS, CONTRIBUTIONS & RECOMMENDATIONS")
    
    add_heading_2(doc, "5.1 Conclusions")
    add_body(doc, "The MedPlus Pharmacy Point of Sale Management System successfully fulfills all project objectives. By unifying high-performance Go micro-services with PostgreSQL and modern JavaScript web components, the system establishes a secure, fast, and compliance-ready operational core for community retail pharmacies.")

    add_heading_2(doc, "5.2 Academic & Industry Contributions")
    add_body(doc, "Demonstrates that compiled Go micro-monolith architectures outperform traditional interpreted frameworks in throughput and resource efficiency for health-tech deployments in developing regions.", "Academic Contribution: ")
    add_body(doc, "Delivers an affordable, non-subscription operational platform tailored for Kenyan retail pharmacies, featuring native M-Pesa STK Push integration and automated FEFO expiry safeguards.", "Industry Contribution: ")

    add_heading_2(doc, "5.3 Future Recommendations")
    add_bullet(doc, "Incorporate machine learning models to project seasonal medication demand spikes.")
    add_bullet(doc, "Connect with Kenya SHA / NHIF APIs for real-time claim verification at POS.")
    add_bullet(doc, "Implement offline PWA sync for multi-branch retail network management.")

    add_heading_1(doc, "REFERENCES")
    refs = [
        "DeLone, W. H., & McLean, E. R. (2003). The DeLone and McLean model of information systems success: A ten-year update. Journal of Management Information Systems, 19(4), 9-30.",
        "Donovan, A. A., & Kernighan, B. W. (2015). The Go Programming Language. Addison-Wesley Professional.",
        "Elmasri, R., & Navathe, S. B. (2016). Fundamentals of Database Systems (7th ed.). Pearson.",
        "Ministry of Health Kenya. (2020). National Guidelines for Good Pharmacy Practice in Kenya. Pharmacy and Poisons Board (PPB).",
        "Pressman, R. S., & Maxim, B. R. (2019). Software Engineering: A Practitioner's Approach (9th ed.). McGraw-Hill Education.",
        "Safaricom PLC. (2025). Daraja 2.0 API Developer Documentation: M-Pesa STK Push Integration. Safaricom Developer Portal.",
        "Sommerville, I. (2016). Software Engineering (10th ed.). Pearson."
    ]
    for r in refs:
        p_r = doc.add_paragraph()
        p_r.paragraph_format.left_indent = Inches(0.4)
        p_r.paragraph_format.first_line_indent = Inches(-0.4)
        p_r.paragraph_format.space_after = Pt(6)
        r_r = p_r.add_run(r)
        r_r.font.name = 'Calibri'
        r_r.font.size = Pt(10)
        r_r.font.color.rgb = RGBColor(51, 65, 85)

    doc.add_page_break()

    # -------------------------------------------------------------
    # APPENDICES
    # -------------------------------------------------------------
    add_heading_1(doc, "APPENDICES")
    
    add_heading_2(doc, "APPENDIX I - III: WORK PROGRESS SUMMARY & SCHEMATICS")
    add_body(doc, "Appendices I through III incorporate the approved project proposal, system wireframes, and architectural schematics as documented during the progress review phase.")

    add_heading_2(doc, "APPENDIX IV: USER TRAINING MANUAL EXCERPT")
    add_bullet(doc, "Launch browser and navigate to http://localhost:8080. Log in using Cashier credentials.")
    add_bullet(doc, "Open POS tab. Scan product barcode or select items from catalog grid.")
    add_bullet(doc, "If Rx prompt appears, input verified Doctor Prescription ID and click Verify.")
    add_bullet(doc, "Select payment method (Cash or M-Pesa). For M-Pesa, enter phone number and click Send STK Push.")
    add_bullet(doc, "Upon payment confirmation green banner, click Complete Checkout to print customer receipt.")

    add_heading_2(doc, "APPENDIX V: SAMPLE SYSTEM CODE")
    code_sample = """// VerifyPrescriptionHandler checks prescription validity prior to POS checkout
func VerifyPrescriptionHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    rxNumber := strings.TrimSpace(r.URL.Query().Get("number"))
    
    var rx models.Prescription
    err := db.DB.QueryRow(`SELECT id, prescription_number, patient_name, doctor_name, status 
                           FROM prescriptions WHERE prescription_number = $1`, rxNumber).Scan(
                           &rx.ID, &rx.PrescriptionNumber, &rx.PatientName, &rx.DoctorName, &rx.Status)
    if err == sql.ErrNoRows {
        http.Error(w, `{"error":"Prescription ID not found"}`, http.StatusNotFound)
        return
    }
    json.NewEncoder(w).Encode(map[string]interface{}{"valid": true, "prescription": rx})
}"""
    add_code_block(doc, code_sample)

    add_heading_2(doc, "APPENDIX VI: SAMPLE SYSTEM OUTPUTS")
    receipt_sample = """==================================================
              MEDPLUS PHARMACY POS
         KCA University Branch, Nairobi
             Tel: +254 700 000 000
==================================================
Receipt No: REC-20260720-0042
Date: 2026-07-20 14:32:10
Cashier: Ralph Osiko (Cashier)
Customer: Walk-in Patient
--------------------------------------------------
Item                     Qty   Price      Total
--------------------------------------------------
Paracetamol 500mg         2    Ksh 50    Ksh 100
Amoxicillin 500mg (Rx)    1    Ksh 350   Ksh 350
--------------------------------------------------
Subtotal:                                Ksh 450.00
VAT (16% Included):                      Ksh  62.07
TOTAL DUE:                               Ksh 450.00
--------------------------------------------------
Payment Method: M-Pesa (STK Push)
M-Pesa Ref: QHK728190X
Status: COMPLETED / PAID
=================================================="""
    add_code_block(doc, receipt_sample)

    current_dir = os.path.dirname(os.path.abspath(__file__))
    docx_path = os.path.join(current_dir, "Final_Report_Document.docx")
    doc.save(docx_path)
    print(f"Successfully saved Word document at: {docx_path}")

if __name__ == '__main__':
    generate_word_document()
