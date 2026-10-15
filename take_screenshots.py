import os
import subprocess

def capture_screens():
    edge = r"C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe"
    if not os.path.exists(edge):
        edge = r"C:\Program Files\Google\Chrome\Application\chrome.exe"

    project_dir = os.path.dirname(os.path.abspath(__file__))
    
    files = [
        ("pharmacy_pos.html", "pos_screen.png"),
        ("implementation_plan.html", "imp_screen.png"),
        ("user_manual.html", "user_screen.png")
    ]

    for html_fname, img_fname in files:
        html_path = os.path.join(project_dir, html_fname)
        img_path = os.path.join(project_dir, img_fname)
        if os.path.exists(html_path):
            cmd = [
                edge,
                "--headless=new",
                "--disable-gpu",
                "--window-size=1280,800",
                f"--screenshot={img_path}",
                f"file:///{html_path.replace('\\', '/')}"
            ]
            res = subprocess.run(cmd, capture_output=True, text=True)
            print(f"Captured {img_fname}: exists={os.path.exists(img_path)}, code={res.returncode}")

if __name__ == '__main__':
    capture_screens()
