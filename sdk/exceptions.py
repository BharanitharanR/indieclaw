class RustGatewayError(Exception):
    pass


class GatewayConnectionError(RustGatewayError):
    pass


class GatewayResponseError(RustGatewayError):
    pass