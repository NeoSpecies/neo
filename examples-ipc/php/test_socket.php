<?php
// 尝试动态加载扩展
if (!extension_loaded('sockets')) {
    echo "Attempting to load sockets extension...\n";
    
    // Windows 系统
    if (strtoupper(substr(PHP_OS, 0, 3)) === 'WIN') {
        // 尝试不同的路径
        $paths = [
            'C:\\Program Files\\Php\\ext\\php_sockets.dll',
            'php_sockets.dll',
            'sockets'
        ];
        
        foreach ($paths as $path) {
            if (@dl($path)) {
                echo "Successfully loaded from: $path\n";
                break;
            }
        }
    }
}

if (extension_loaded('sockets')) {
    echo "✓ Sockets extension is now loaded!\n";
    
    // 测试创建 socket
    $socket = socket_create(AF_INET, SOCK_STREAM, SOL_TCP);
    if ($socket !== false) {
        echo "✓ Socket creation successful\n";
        socket_close($socket);
    }
} else {
    echo "✗ Failed to load sockets extension\n";
    echo "\nPlease check:\n";
    echo "1. Is php.ini correctly configured?\n";
    echo "2. Did you restart your command prompt after editing php.ini?\n";
    echo "3. Is the extension_dir path correct in php.ini?\n";
}
?>