<?php
// Image Verification Script
// This script checks if your social media images are accessible and properly configured

header('Content-Type: text/html; charset=utf-8');
echo "<!DOCTYPE html>
<html lang='en'>
<head>
    <meta charset='UTF-8'>
    <meta name='viewport' content='width=device-width, initial-scale=1.0'>
    <title>Social Media Image Verification</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .test { margin: 20px 0; padding: 15px; border: 1px solid #ddd; border-radius: 5px; }
        .success { background-color: #d4edda; border-color: #c3e6cb; }
        .error { background-color: #f8d7da; border-color: #f5c6cb; }
        .warning { background-color: #fff3cd; border-color: #ffeaa7; }
        table { border-collapse: collapse; width: 100%; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #f2f2f2; }
        img { max-width: 200px; border: 1px solid #ccc; }
    </style>
</head>
<body>
    <h1>Social Media Image Verification</h1>
    <p>This tool verifies your social media images are properly configured and accessible.</p>
";

$images = [
    'aura-genesis-social.jpg',
    'apex-os-lq-social.jpg',
    'foundation-social.jpg',
    'icf-social.jpg'
];

$baseUrl = 'https://' . $_SERVER['HTTP_HOST'] . '/images/';

echo "<h2>Image Accessibility Test</h2>";
foreach ($images as $image) {
    $url = $baseUrl . $image;
    echo "<div class='test'>";

    // Test HTTP response
    $headers = @get_headers($url);
    if ($headers && strpos($headers[0], '200') !== false) {
        echo "<div class='success'><strong>✅ $image</strong> - Accessible</div>";
        echo "<img src='$url' alt='$image' style='display: block; margin: 10px 0;'>";

        // Check content type
        $contentType = '';
        foreach ($headers as $header) {
            if (stripos($header, 'content-type:') === 0) {
                $contentType = trim(str_replace('content-type:', '', $header));
                break;
            }
        }

        if (strpos($contentType, 'image/jpeg') === false) {
            echo "<div class='warning'>⚠️ Warning: Content-Type is '$contentType', should be 'image/jpeg'</div>";
        }

        // Get file size
        $fileSize = 0;
        foreach ($headers as $header) {
            if (stripos($header, 'content-length:') === 0) {
                $fileSize = (int) trim(str_replace('content-length:', '', $header));
                break;
            }
        }

        if ($fileSize > 0) {
            $sizeMB = round($fileSize / 1024 / 1024, 2);
            echo "<div>File size: {$sizeMB}MB</div>";
            if ($sizeMB > 5) {
                echo "<div class='warning'>⚠️ Warning: File size > 5MB may cause issues</div>";
            }
        }

    } else {
        echo "<div class='error'><strong>❌ $image</strong> - Not accessible</div>";
        echo "<div>URL: <a href='$url' target='_blank'>$url</a></div>";
        if ($headers) {
            echo "<div>Response: " . $headers[0] . "</div>";
        } else {
            echo "<div>Response: No response from server</div>";
        }
    }

    echo "</div>";
}

echo "
<h2>Meta Tag Verification</h2>
<div class='test'>
    <p>Check that your meta tags reference the correct image URLs:</p>
    <ul>
        <li><strong>Homepage</strong>: https://aura-genesis.org/images/aura-genesis-social.jpg</li>
        <li><strong>APEX-OS-LQ</strong>: https://aura-genesis.org/images/apex-os-lq-social.jpg</li>
        <li><strong>Foundation</strong>: https://aura-genesis.org/images/foundation-social.jpg</li>
        <li><strong>ICF</strong>: https://aura-genesis.org/images/icf-social.jpg</li>
    </ul>
</div>

<h2>Next Steps</h2>
<div class='test'>
    <ol>
        <li>If images are not accessible, check file upload and permissions</li>
        <li>Test social sharing using the debuggers listed in troubleshooting.md</li>
        <li>Clear social media caches if images were recently updated</li>
        <li>Contact your hosting provider if server issues persist</li>
    </ol>
</div>

<h2>Debug Tools</h2>
<div class='test'>
    <ul>
        <li><a href='test.html' target='_blank'>Image Display Test</a></li>
        <li><a href='troubleshooting.md' target='_blank'>Full Troubleshooting Guide</a></li>
        <li><a href='https://developers.facebook.com/tools/debug/' target='_blank'>Facebook Debugger</a></li>
        <li><a href='https://cards-dev.twitter.com/validator' target='_blank'>Twitter Validator</a></li>
    </ul>
</div>

</body>
</html>";
?>
