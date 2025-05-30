import threading
from http.server import HTTPServer, BaseHTTPRequestHandler
from prometheus_client import generate_latest, CONTENT_TYPE_LATEST
from typing import Optional
import logging

logger = logging.getLogger(__name__)

class MetricsHandler(BaseHTTPRequestHandler):
    """处理 Prometheus 指标请求的 HTTP 处理器"""
    
    def do_GET(self):
        """处理 GET 请求"""
        if self.path == '/metrics':
            self.send_response(200)
            self.send_header('Content-Type', CONTENT_TYPE_LATEST)
            self.end_headers()
            self.wfile.write(generate_latest())
        else:
            self.send_response(404)
            self.end_headers()
            self.wfile.write(b'Not Found')

    def log_message(self, format: str, *args):
        """重写日志方法，使用标准日志记录器"""
        logger.info(f"{self.address_string()} - {format%args}")

class MetricsServer:
    """Prometheus 指标暴露服务器"""
    
    def __init__(self, host: str = 'localhost', port: int = 8000):
        self.host = host
        self.port = port
        self._server: Optional[HTTPServer] = None
        self._thread: Optional[threading.Thread] = None
        self._stopped = False

    def start(self):
        """启动指标服务器"""
        if self._server is not None:
            return

        try:
            self._server = HTTPServer((self.host, self.port), MetricsHandler)
            self._thread = threading.Thread(
                target=self._run_server,
                name="MetricsServer",
                daemon=True
            )
            self._thread.start()
            logger.info(f"Metrics server started at http://{self.host}:{self.port}/metrics")
        except Exception as e:
            logger.error(f"Failed to start metrics server: {e}")
            self._server = None
            raise

    def _run_server(self):
        """运行服务器的线程函数"""
        while not self._stopped:
            try:
                self._server.serve_forever()
            except Exception as e:
                if not self._stopped:
                    logger.error(f"Metrics server error: {e}")
                    continue
                break

    def stop(self):
        """停止指标服务器"""
        self._stopped = True
        if self._server is not None:
            self._server.shutdown()
            self._server.server_close()
            self._server = None
        
        if self._thread is not None:
            self._thread.join()
            self._thread = None
        
        logger.info("Metrics server stopped") 