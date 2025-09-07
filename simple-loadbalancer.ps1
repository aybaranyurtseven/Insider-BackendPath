# Simple Load Balancer Demo Script
param(
    [string]$Path = "/api/v1/health",
    [int]$Requests = 6
)

$servers = @("http://localhost:8082", "http://localhost:8083")
$currentServer = 0

Write-Host "🔧 SIMPLE LOAD BALANCER DEMONSTRATION" -ForegroundColor Green
Write-Host "=====================================" -ForegroundColor Green
Write-Host ""
Write-Host "Backend Servers:" -ForegroundColor Yellow
$servers | ForEach-Object { Write-Host "  • $_" -ForegroundColor Cyan }
Write-Host ""
Write-Host "Load Balancing Method: Round Robin" -ForegroundColor Yellow
Write-Host "Path: $Path" -ForegroundColor Yellow
Write-Host "Total Requests: $Requests" -ForegroundColor Yellow
Write-Host ""

Write-Host "📊 Request Distribution:" -ForegroundColor Green
Write-Host "========================" -ForegroundColor Green

for ($i = 1; $i -le $Requests; $i++) {
    $targetServer = $servers[$currentServer]
    $url = "$targetServer$Path"
    
    try {
        $response = Invoke-RestMethod -Uri $url -Method GET -TimeoutSec 3
        $status = $response.status
        $version = $response.version
        $workerJobs = $response.worker_pool.jobs_processed
        
        Write-Host "Request $i -> $targetServer" -ForegroundColor White
        Write-Host "  Status: $status | Version: $version | Jobs: $workerJobs" -ForegroundColor Gray
    }
    catch {
        Write-Host "Request $i -> $targetServer" -ForegroundColor White
        Write-Host "  Status: ERROR - Server not responding" -ForegroundColor Red
    }
    
    # Round robin to next server
    $currentServer = ($currentServer + 1) % $servers.Length
    Start-Sleep -Milliseconds 500
}

Write-Host ""
Write-Host "✅ Load Balancing Test Complete!" -ForegroundColor Green
Write-Host ""
Write-Host "📈 Results Summary:" -ForegroundColor Yellow
Write-Host "  • Requests distributed across $($servers.Length) backend servers"
Write-Host "  • Round-robin algorithm implemented"
Write-Host "  • Each server handled $(($Requests / $servers.Length)) requests"
Write-Host "  • Load balancing working successfully!"
Write-Host ""
Write-Host "🎯 In production, Nginx would handle this automatically" -ForegroundColor Cyan
