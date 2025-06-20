import pytest
import asyncio
from unittest.mock import Mock, AsyncMock
from discovery import ServiceDiscovery, ServiceInfo
from health import HealthChecker, HealthCheck, HealthStatus
from registrar import ServiceRegistrar

@pytest.fixture
def mock_etcd_client():
    """Mock etcd client"""
    client = Mock()
    client.lease = AsyncMock()
    client.put = AsyncMock(return_value=True)
    client.delete = AsyncMock(return_value=True)
    client.get_prefix = AsyncMock(return_value=[])
    client.watch_prefix = Mock()
    client.refresh_lease = AsyncMock()
    return client

@pytest.fixture
def discovery(mock_etcd_client):
    """Service discovery fixture"""
    discovery = ServiceDiscovery()
    discovery.client = mock_etcd_client
    return discovery

@pytest.fixture
def health_checker():
    """Health checker fixture"""
    return HealthChecker()

@pytest.fixture
def registrar(discovery, health_checker):
    """Service registrar fixture"""
    return ServiceRegistrar(discovery, health_checker)

@pytest.mark.asyncio
async def test_service_registration(discovery):
    """Test service registration"""
    service = ServiceInfo(
        name="test-service",
        id="test-1",
        host="localhost",
        port=8080,
        metadata={"version": "1.0.0"},
        version="1.0.0",
        weight=100
    )
    
    success = await discovery.register_service(service)
    assert success
    
    discovery.client.put.assert_called_once()
    assert "/services/test-service/test-1" in discovery.client.put.call_args[0]

@pytest.mark.asyncio
async def test_service_deregistration(discovery):
    """Test service deregistration"""
    service = ServiceInfo(
        name="test-service",
        id="test-1",
        host="localhost",
        port=8080,
        metadata={},
        version="1.0.0",
        weight=100
    )
    
    success = await discovery.deregister_service(service)
    assert success
    
    discovery.client.delete.assert_called_once()
    assert "/services/test-service/test-1" in discovery.client.delete.call_args[0]

@pytest.mark.asyncio
async def test_service_discovery(discovery):
    """Test service discovery"""
    # Mock service data
    service_data = {
        "name": "test-service",
        "id": "test-1",
        "host": "localhost",
        "port": 8080,
        "metadata": {},
        "version": "1.0.0",
        "weight": 100
    }
    
    # Setup mock response
    discovery.client.get_prefix.return_value = [
        (bytes(str(service_data), 'utf-8'), None)
    ]
    
    services = await discovery.discover_service("test-service")
    assert len(services) == 1
    assert services[0].name == "test-service"
    assert services[0].id == "test-1"

@pytest.mark.asyncio
async def test_health_checker():
    """Test health checker"""
    checker = HealthChecker()
    
    # Mock health check function
    check_result = {"status": "ok"}
    async def check_func():
        return check_result
    
    # Add check
    checker.add_check(
        "test-check",
        check_func,
        HealthCheck(interval=0.1)
    )
    
    # Wait for check to run
    await asyncio.sleep(0.2)
    
    # Get result
    result = checker.get_result("test-check")
    assert result is not None
    assert result.status == HealthStatus.HEALTHY
    
    # Cleanup
    checker.close()

@pytest.mark.asyncio
async def test_service_registrar(registrar):
    """Test service registrar"""
    # Mock health check
    async def check_func():
        return {"status": "ok"}
    
    # Register service
    service_id = await registrar.register(
        name="test-service",
        host="localhost",
        port=8080,
        checks=[
            {
                "name": "test-check",
                "check_func": check_func
            }
        ]
    )
    
    assert service_id is not None
    assert service_id in registrar._registered_services
    
    # Deregister service
    success = await registrar.deregister(service_id)
    assert success
    assert service_id not in registrar._registered_services
    
    # Cleanup
    registrar.close() 