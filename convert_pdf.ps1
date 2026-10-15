$edge = "C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe"
if (-not (Test-Path $edge)) {
    $edge = "C:\Program Files\Google\Chrome\Application\chrome.exe"
}
$html = Join-Path $PSScriptRoot "final_report.html"
$pdf = Join-Path $PSScriptRoot "Final_Report_Document.pdf"

$argList = @(
    "--headless",
    "--disable-gpu",
    "--no-pdf-header-footer",
    "--print-to-pdf=$pdf",
    "$html"
)

Write-Host "Running $edge ..."
$proc = Start-Process -FilePath $edge -ArgumentList $argList -Wait -PassThru
Write-Host "Exit code: $($proc.ExitCode)"

if (Test-Path $pdf) {
    $item = Get-Item $pdf
    Write-Host "PDF generated successfully! Size: $($item.Length) bytes"
} else {
    Write-Host "PDF generation failed."
}
