$word = New-Object -ComObject Word.Application
$word.Visible = $false
$inputDoc = Join-Path $PSScriptRoot "Final Report Document-Guide.doc"
$outputTxt = Join-Path $PSScriptRoot "final_report_guide_extracted.txt"
$doc = $word.Documents.Open($inputDoc)
$text = $doc.Content.Text
Set-Content -Path $outputTxt -Value $text -Encoding UTF8
$doc.Close()
$word.Quit()
