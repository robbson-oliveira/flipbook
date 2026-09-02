$pdftoppm = "C:\Users\LENOVO\AppData\Local\Microsoft\WinGet\Packages\oschwartz10612.Poppler_Microsoft.Winget.Source_8wekyb3d8bbwe\poppler-25.07.0\Library\bin\pdftoppm.exe"
$srcPdf = (Get-ChildItem "F:\Websites\flipbook\*.pdf" | Select-Object -First 1).FullName
$asciiPdf = "F:\Websites\flipbook\adec2025.pdf"

if (!(Test-Path $asciiPdf)) {
    Copy-Item -Path $srcPdf -Destination $asciiPdf
}

$pdfPath = $asciiPdf
$outDir = "F:\Websites\flipbook\pages_hd"

if (!(Test-Path $outDir)) {
    New-Item -ItemType Directory -Force -Path $outDir | Out-Null
}

Write-Host "Starting extraction of all 25 pages at Ultra HD 300 DPI..."
& $pdftoppm -png -r 300 -f 1 -l 25 $pdfPath "$outDir\page"

Write-Host "Copying pages to /pages and /web/pages..."
Get-ChildItem "$outDir\page-*.png" | ForEach-Object {
    $num = $_.BaseName -replace '^page-0*',''
    $formattedName = "page-" + $num.PadLeft(2, '0') + ".png"
    
    Copy-Item -Path $_.FullName -Destination "F:\Websites\flipbook\pages\$formattedName" -Force
    Copy-Item -Path $_.FullName -Destination "F:\Websites\flipbook\web\pages\$formattedName" -Force
}

$total = (Get-ChildItem "F:\Websites\flipbook\pages\page-*.png").Count
Write-Host "Conversion completed! Total pages in HD: $total"
