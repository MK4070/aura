from httpx import ASGITransport, AsyncClient

import app.dependencies as deps
from app import app


async def test_lifespan_initialization() -> None:
    """
    Integration test to verify that the FastAPI lifespan context manager
    correctly initializes our global infrastructure clients in the dependency
    module before accepting traffic.
    """
    async with app.router.lifespan_context(app):
        # Verify the underlying network clients were instantiated
        assert deps._httpx_client is not None
        assert deps._qdrant_client is not None

    # Exiting the 'async with' block triggers the lifespan shutdown event,
    # ensuring our .aclose() and .close() methods are called to
    # prevent connection leaks.


# async def test_cors_middleware():
#     """
#     Verifies that the global CORS middleware is correctly intercepting
#     preflight OPTIONS requests and returning the proper headers.
#     """
#     transport = ASGITransport(app=app)
#     async with AsyncClient(transport=transport, base_url="http://testserver") as c:
#         # Simulate a browser preflight request from a separate frontend domain
#         response = await c.options(
#             "/api/v1/query",
#             headers={
#                 "Origin": "http://localhost:3000",
#                 "Access-Control-Request-Method": "POST",
#             },
#         )

#         assert response.status_code == 200
#         assert "access-control-allow-origin" in response.headers


async def test_api_router_prefix() -> None:
    """
    Verifies that the application routers are mounted to the correct
    API version prefixes (e.g., /api/v1).
    """
    transport = ASGITransport(app=app)
    async with AsyncClient(transport=transport, base_url="http://testserver") as client:
        response_404 = await client.post("/query", json={"query": "test", "top_k": 1})
        assert response_404.status_code == 404

        # We don't test the actual logic here (that's in test_routes.py),
        # just that the routing prefix exists and returns a validation error (422)
        # instead of a 404 Not Found.
        response_422 = await client.post("/api/v1/query", json={})
        assert response_422.status_code == 422
