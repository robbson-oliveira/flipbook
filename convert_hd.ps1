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

Write-Host "Extracting pages 1 to 4 at 300 DPI..."
# -png : output PNG
# -f 1 -l 4 : pages 1 to 4
# -r 300 : 300 DPI (crisp, high-definition)
& $pdftoppm -png -r 300 -f 1 -l 4 $pdfPath "$outDir\page"

Write-Host "Checking extracted files:"
Get-ChildItem $outDir | ForEach-Object {
    $img = [System.Drawing.Image]::FromFile($_.FullName)
    Write-Host "$($_.Name) -> $($img.Width)x$($img.Height) px, $([math]::Round($_.Length/1MB, 2)) MB"
    $img.Dispose()
}
