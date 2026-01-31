<?php
// APEX-OS-LQ Downloads API
// This is a basic PHP endpoint for serving download information
// In production, this would be enhanced with authentication, rate limiting, etc.

header('Content-Type: application/json');
header('Access-Control-Allow-Origin: *');
header('Access-Control-Allow-Methods: GET, POST, OPTIONS');
header('Access-Control-Allow-Headers: Content-Type');

// Handle preflight requests
if ($_SERVER['REQUEST_METHOD'] === 'OPTIONS') {
    exit(0);
}

// Load manifest data
$manifestPath = __DIR__ . '/../downloads/manifest.json';
if (!file_exists($manifestPath)) {
    http_response_code(500);
    echo json_encode(['error' => 'Manifest not found']);
    exit;
}

$manifest = json_decode(file_get_contents($manifestPath), true);
if ($manifest === null) {
    http_response_code(500);
    echo json_encode(['error' => 'Invalid manifest format']);
    exit;
}

// Handle different API endpoints
$requestUri = $_SERVER['REQUEST_URI'];
$path = parse_url($requestUri, PHP_URL_PATH);

// Remove base path if needed
$path = str_replace('/api', '', $path);

switch ($path) {
    case '/downloads':
    case '/downloads/':
        // Return full manifest
        echo json_encode($manifest);
        break;

    case '/downloads/platforms':
        // Return available platforms
        $platforms = array_keys($manifest['packages']);
        echo json_encode([
            'platforms' => $platforms,
            'total' => count($platforms)
        ]);
        break;

    case '/downloads/latest':
        // Return latest version info
        echo json_encode([
            'version' => $manifest['version'],
            'release_date' => $manifest['release_date'],
            'packages' => $manifest['packages']
        ]);
        break;

    default:
        // Check if requesting specific platform
        $platform = str_replace('/downloads/', '', $path);
        if (isset($manifest['packages'][$platform])) {
            echo json_encode($manifest['packages'][$platform]);
        } else {
            http_response_code(404);
            echo json_encode(['error' => 'Platform not found']);
        }
        break;
}

// Log API access (basic logging)
$logFile = __DIR__ . '/../logs/api_access.log';
$logEntry = date('Y-m-d H:i:s') . ' - ' . $_SERVER['REMOTE_ADDR'] . ' - ' . $_SERVER['REQUEST_METHOD'] . ' - ' . $requestUri . "\n";
file_put_contents($logFile, $logEntry, FILE_APPEND | LOCK_EX);
?>
