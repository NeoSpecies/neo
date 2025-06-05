def call_async(self, method: str, params: dict, callback: callable):
    msg_id = str(uuid.uuid4())
    # 在协议头设置回调标识
    header = ProtocolHeader()
    header.callback_flag = 1
    header.callback_id_len = len(msg_id)
    
    # 注册回调
    self.async_bridge.register_callback(msg_id, callback)
    
    # 发送带回调标识的请求
    self._send_request(header, msg_id, method, params)