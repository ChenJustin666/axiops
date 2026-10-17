param(
    [string]$BaseUrl = 'http://127.0.0.1:3006/api'
)

$ErrorActionPreference = 'Stop'

function Login([string]$Username, [string]$Password) {
    $json = @{ username = $Username; password = $Password } | ConvertTo-Json
    $response = Invoke-RestMethod -Method Post `
        -Uri "$BaseUrl/platform/v1/auth/login" `
        -ContentType 'application/json; charset=utf-8' `
        -Body ([Text.Encoding]::UTF8.GetBytes($json))
    return $response.data.access_token
}

function Get-Authorized([string]$Path, [string]$Token) {
    return Invoke-RestMethod -Uri "$BaseUrl$Path" -Headers @{ Authorization = "Bearer $Token" }
}

function Assert-EqualSet([string]$Name, [object[]]$Actual, [string[]]$Expected) {
    $actualText = (@($Actual) | Sort-Object) -join ','
    $expectedText = (@($Expected) | Sort-Object) -join ','
    if ($actualText -ne $expectedText) {
        throw "$Name failed: expected [$expectedText], actual [$actualText]"
    }
}

function Assert-Forbidden([string]$Path, [string]$Token) {
    try {
        Get-Authorized $Path $Token | Out-Null
        throw "$Path should return HTTP 403"
    } catch {
        if (-not $_.Exception.Response -or [int]$_.Exception.Response.StatusCode -ne 403) {
            throw
        }
    }
}

$cases = @(
    @{ User = 'admin'; Password = 'REPLACE_ME'; Projects = @('axiops-demo-yaml', 'axiops-demo-helm', 'axiops-demo-host'); AccessMenu = $true },
    @{ User = 'axiops_dev'; Password = 'REPLACE_ME'; Projects = @('axiops-demo-yaml'); AccessMenu = $false },
    @{ User = 'axiops_test'; Password = 'REPLACE_ME'; Projects = @('axiops-demo-helm'); AccessMenu = $false },
    @{ User = 'axiops_ops'; Password = 'REPLACE_ME'; Projects = @('axiops-demo-yaml', 'axiops-demo-helm', 'axiops-demo-host'); AccessMenu = $false }
)

foreach ($case in $cases) {
    $token = Login $case.User $case.Password
    $projects = Get-Authorized '/platform/v1/projects?page=1&page_size=100' $token
    $menus = Get-Authorized '/platform/v1/auth/menus' $token
    $projectCodes = @($projects.data.list | ForEach-Object { $_.code })
    $menuCodes = @($menus.data | ForEach-Object { $_.code })

    Assert-EqualSet "$($case.User) project scope" $projectCodes $case.Projects
    $hasAccessMenu = $menuCodes -contains 'system.access'
    if ($hasAccessMenu -ne $case.AccessMenu) {
        throw "$($case.User) permission-center menu mismatch"
    }
    Write-Host "PASS $($case.User): projects=$($projectCodes -join ',')"
}

$developerToken = Login 'axiops_dev' 'REPLACE_ME'
Assert-Forbidden '/platform/v1/projects/2' $developerToken
Assert-Forbidden '/cicd/v1/services/3' $developerToken
Write-Host 'PASS developer cross-project requests return HTTP 403'
Write-Host 'AxiOps authorization acceptance passed.'
