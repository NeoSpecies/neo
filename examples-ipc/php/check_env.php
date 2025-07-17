<?php
echo "=== PHP Environment Check for Neo Framework ===\n\n";

// Check PHP version
echo "1. PHP Version: " . PHP_VERSION . "\n";
if (version_compare(PHP_VERSION, '7.0.0', '>=')) {
    echo "   ✓ PHP version is compatible\n";
} else {
    echo "   ✗ PHP version is too old. Need PHP 7.0+\n";
}
echo "\n";

// Check sockets extension
echo "2. Sockets Extension:\n";
if (extension_loaded('sockets')) {
    echo "   ✓ Sockets extension is loaded\n";
    
    // Test socket functions
    $testSocket = @socket_create(AF_INET, SOCK_STREAM, SOL_TCP);
    if ($testSocket !== false) {
        echo "   ✓ Socket functions are working\n";
        socket_close($testSocket);
    } else {
        echo "   ✗ Socket functions are not working\n";
    }
} else {
    echo "   ✗ Sockets extension is NOT loaded\n";
    echo "   Please enable it in php.ini:\n";
    echo "   - For Windows: extension=sockets or extension=php_sockets.dll\n";
    echo "   - For Linux/Mac: extension=sockets.so\n";
}
echo "\n";

// Check JSON extension (usually built-in)
echo "3. JSON Extension:\n";
if (extension_loaded('json')) {
    echo "   ✓ JSON extension is loaded\n";
} else {
    echo "   ✗ JSON extension is NOT loaded\n";
}
echo "\n";

// Show php.ini location
echo "4. PHP Configuration:\n";
echo "   php.ini location: " . php_ini_loaded_file() . "\n";
echo "   Additional .ini files: " . php_ini_scanned_files() . "\n";
echo "\n";

// Show all loaded extensions
echo "5. All Loaded Extensions:\n";
$extensions = get_loaded_extensions();
echo "   " . implode(', ', $extensions) . "\n";
echo "\n";

// Test connectivity
echo "6. Network Test:\n";
echo "   Testing connection to localhost:9999...\n";
$socket = @socket_create(AF_INET, SOCK_STREAM, SOL_TCP);
if ($socket !== false) {
    $result = @socket_connect($socket, 'localhost', 9999);
    if ($result) {
        echo "   ✓ Can connect to IPC server\n";
    } else {
        echo "   ✗ Cannot connect to IPC server (is Neo gateway running?)\n";
    }
    socket_close($socket);
} else {
    echo "   ✗ Cannot create socket\n";
}

echo "\n=== Check Complete ===\n";