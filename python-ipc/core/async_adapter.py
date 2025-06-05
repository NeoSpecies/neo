import uuid
from typing import Callable, Any

class AsyncBridge:
    def __init__(self):
        self.pending_callbacks = {}
        
    def register_callback(self, msg_id: str, callback: Callable[[Any, Exception], None]):
        self.pending_callbacks[msg_id] = {
            'callback': callback,
            'timestamp': time.time()
        }
    
    def handle_response(self, msg_id: str, response: Any, error: str):
        entry = self.pending_callbacks.pop(msg_id, None)
        if entry and entry['callback']:
            exc = Exception(error) if error else None
            entry['callback'](response, exc)