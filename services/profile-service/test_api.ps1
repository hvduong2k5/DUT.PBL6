$cases = @(
    "p. thuan hoa tp hue",
    "vi da",
    "phu bai huong thuy",
    "kim long",
    "thuy xuan lang huong",
    "xa vinh loc"
)

Write-Host "=== TEST HE THONG PROFILE SERVICE TREN DOCKER ===" -ForegroundColor Cyan

foreach ($c in $cases) {
    $payload = @{ raw_address = $c } | ConvertTo-Json
    $res = Invoke-RestMethod -Uri "http://localhost:8080/api/v1/profile/addresses/validate" -Method Post -ContentType "application/json" -Body $payload
    Write-Host "Input: '$c' => Match: $($res.Code) | $($res.Name) | Score: $($res.Score)" -ForegroundColor Green
}
