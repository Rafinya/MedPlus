$content = Get-Content 'pharmacy_pos.html' -Raw
$styleMatch = [regex]::Match($content, '<style>(.*?)</style>', [System.Text.RegularExpressions.RegexOptions]::Singleline)
$scriptMatch = [regex]::Match($content, '<script>(.*?)</script>\s*</body>', [System.Text.RegularExpressions.RegexOptions]::Singleline)
$bodyMatch = [regex]::Match($content, '<body>(.*?)<script>', [System.Text.RegularExpressions.RegexOptions]::Singleline)
if (-not $styleMatch.Success -or -not $scriptMatch.Success -or -not $bodyMatch.Success) {
    throw 'Could not split frontend file.'
}
$style = $styleMatch.Groups[1].Value.Trim()
$script = $scriptMatch.Groups[1].Value.Trim()
$body = $bodyMatch.Groups[1].Value.Trim()
$index = @"
<!doctype html>
<html lang="en">
<head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>MedPlus Pharmacy POS</title>
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&display=swap" rel="stylesheet" />
    <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.4.0/css/all.min.css" />
    <link rel="stylesheet" href="/assets/app.css" />
</head>
<body>
$body
<script src="/assets/app.js" defer></script>
</body>
</html>
"@
Set-Content 'web/assets/app.css' $style
Set-Content 'web/assets/app.js' $script
Set-Content 'web/index.html' $index
