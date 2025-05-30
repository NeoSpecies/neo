class ProtocolError(Exception):
    """Base class for protocol errors"""
    pass

class InvalidMessageError(ProtocolError):
    """Invalid message format"""
    pass

class ChecksumMismatchError(ProtocolError):
    """Checksum verification failed"""
    pass

class MessageTooLargeError(ProtocolError):
    """Message size exceeds limit"""
    pass

class CompressionError(ProtocolError):
    """Base class for compression-related errors"""
    pass

class CompressionFailedError(CompressionError):
    """Compression operation failed"""
    pass

class DecompressionFailedError(CompressionError):
    """Decompression operation failed"""
    pass

class InvalidCompressionError(CompressionError):
    """Invalid compression type"""
    pass

class MaxRetryExceededError(ProtocolError):
    """Maximum retry count exceeded"""
    pass 