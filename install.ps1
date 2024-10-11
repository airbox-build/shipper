# Check if running with administrative privileges
$isAdmin = ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)

if (-not $isAdmin) {
    Write-Host "Please run this script as an administrator."
    exit
}

$GH_REPO = "airbox-build/shipper"
$TIMEOUT = 90
$CONFIG_PATH = "C:\ProgramData\AirBox\shipper.yml"

# Parse command line arguments
param(
    [string]$config = $CONFIG_PATH
)

$CONFIG_PATH = $config

# Get the current logged-in user
$USERNAME = $env:USERNAME

$VERSION = (Invoke-WebRequest -Uri "https://api.github.com/repos/$GH_REPO/releases/latest" -UseBasicParsing).Content | ConvertFrom-Json | Select-Object -ExpandProperty tag_name
if (-not $VERSION) {
    Write-Host "`nThere was an error trying to check what is the latest version of airbox shipper.`nPlease try again later.`n"
    exit 1
}

$OS_type = $env:PROCESSOR_ARCHITECTURE
switch ($OS_type) {
    "AMD64", "x86_64" {
        $OS_type = "amd64"
    }
    "x86", "i386" {
        $OS_type = "386"
    }
    "ARM64" {
        $OS_type = "arm64"
    }
    default {
        Write-Host "OS type not supported"
        exit 2
    }
}

$GH_REPO_BIN = "shipper-${VERSION}-windows-${OS_type}.tar.gz"

# Create tmp directory
$TMP_DIR = New-TemporaryFile -Directory | Select-Object -ExpandProperty FullName
Write-Host "Change to temporary directory $TMP_DIR"
Set-Location $TMP_DIR

Write-Host "Downloading AirBox Shipper $VERSION"
$LINK = "https://github.com/$GH_REPO/releases/download/$VERSION/$GH_REPO_BIN"
Write-Host "Downloading $LINK"

Invoke-WebRequest -Uri $LINK -OutFile "$TMP_DIR\$GH_REPO_BIN"
if (-not $?) {
    Write-Host "Error downloading"
    exit 2
}

# Extract and install
$BINARY_PATH = "C:\Program Files\AirBox"
$null = New-Item -Path $BINARY_PATH -ItemType Directory -Force

Expand-Archive -Path "$TMP_DIR\$GH_REPO_BIN" -DestinationPath $BINARY_PATH -Force
if (-not $?) {
    Write-Host "Error extracting files"
    exit 2
}

$BINARY_DIRECTORY = "C:\ProgramData\AirBox"
$null = New-Item -Path $BINARY_DIRECTORY -ItemType Directory -Force

Remove-Item -Path $TMP_DIR -Recurse -Force
Write-Host "Installed successfully to $BINARY_PATH\airbox-shipper.exe"

# Create the service
$SERVICE_NAME = "AirboxShipper"
$SERVICE_PATH = "$BINARY_PATH\airbox-shipper.exe"
$SERVICE_CONFIG_PATH = "$CONFIG_PATH"

$serviceParams = @{
    Name             = $SERVICE_NAME
    BinaryPathName   = "$SERVICE_PATH --config=$SERVICE_CONFIG_PATH"
    DisplayName      = $SERVICE_NAME
    Description      = "AirBox Shipper Service"
    StartupType      = "Automatic"
    Credential       = "LocalSystem"
    DependsOn        = @("tcpip")
    ErrorControl     = "Normal"
}

$service = New-Service @serviceParams -ErrorAction SilentlyContinue
if ($service) {
    Write-Host "Service created successfully."
} else {
    Write-Host "Failed to create the service."
}
